package test

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/gosimple/slug"
	"go.uber.org/zap"
)

var (
	ErrBadTestFile = kilonova.Statusf(400, "Bad test score file")
	ErrBadArchive  = kilonova.Statusf(400, "Bad archive")
)

type archiveTest struct {
	InFile  *zip.File
	OutFile *zip.File
	Score   int
}

type ArchiveCtx struct {
	tests       map[int]archiveTest
	scoredTests []int
	props       *Properties
}

type Subtask struct {
	Score int
	Tests []int
}

type Properties struct {
	Subtasks map[int]Subtask
	// seconds
	TimeLimit *float64
	// kbytes
	MemoryLimit *int

	Author       *string
	Source       *string
	ConsoleInput *bool
	TestName     *string

	DefaultPoints *int

	SubtaskedTests []int
}

func NewArchiveCtx() *ArchiveCtx {
	return &ArchiveCtx{tests: make(map[int]archiveTest), scoredTests: make([]int, 0, 10)}
}

func ProcessScoreFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	f, err := file.Open()
	if err != nil {
		return kilonova.Statusf(500, "Unknown error")
	}
	defer f.Close()

	br := bufio.NewScanner(f)

	for br.Scan() {
		line := br.Text()

		if line == "" { // empty line, skip
			continue
		}

		var testID int
		var score int
		if _, err := fmt.Sscanf(line, "%d %d\n", &testID, &score); err != nil {
			// Might just be a bad line
			continue
		}

		test := ctx.tests[testID]
		test.Score = score
		ctx.tests[testID] = test
		for _, ex := range ctx.scoredTests {
			if ex == testID {
				return ErrBadTestFile
			}
		}

		ctx.scoredTests = append(ctx.scoredTests, testID)
	}
	if br.Err() != nil {
		zap.S().Info(br.Err())
		return kilonova.WrapError(err, "Score file read error")
	}
	return nil
}

func ProcessArchiveFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	name := path.Base(file.Name)
	if strings.HasSuffix(name, ".txt") { // test score file
		return ProcessScoreFile(ctx, file)
	}

	if strings.HasSuffix(name, ".properties") { // test properties file
		return ProcessPropertiesFile(ctx, file)
	}

	// if nothing else is detected, it should be a test file

	var tid int
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		// maybe it's problem_name.%d.{in,sol,out} format
		nm := strings.Split(strings.TrimSuffix(name, path.Ext(name)), ".")
		if len(nm) == 0 {
			return nil
		}
		val, err := strconv.Atoi(nm[len(nm)-1])
		if err != nil {
			return nil
		}
		tid = val
	}

	if strings.HasSuffix(name, ".in") { // test input file
		tf := ctx.tests[tid]
		if tf.InFile != nil { // in file already exists
			return kilonova.Statusf(400, "Multiple input files for test %d", tid)
		}

		tf.InFile = file
		ctx.tests[tid] = tf
	}
	if strings.HasSuffix(name, ".out") || strings.HasSuffix(name, ".ok") || strings.HasSuffix(name, ".sol") { // test output file
		tf := ctx.tests[tid]
		if tf.OutFile != nil { // out file already exists
			return kilonova.Statusf(400, "Multiple output files for test %d", tid)
		}

		tf.OutFile = file
		ctx.tests[tid] = tf
	}
	return nil
}

func ProcessZipTestArchive(ctx context.Context, pb *kilonova.Problem, ar *zip.Reader, base *sudoapi.BaseAPI) *kilonova.StatusError {
	aCtx := NewArchiveCtx()

	for _, file := range ar.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := ProcessArchiveFile(aCtx, file); err != nil {
			return err
		}
	}

	if aCtx.props != nil && aCtx.props.Subtasks != nil && len(aCtx.props.SubtaskedTests) != len(aCtx.tests) {
		zap.S().Info(len(aCtx.props.SubtaskedTests), len(aCtx.tests))
		return kilonova.Statusf(400, "Mismatched number of tests in archive and tests that correspond to at least one subtask")
	}

	for k, v := range aCtx.tests {
		if v.InFile == nil || v.OutFile == nil {
			return kilonova.Statusf(400, "Missing input or output file for test %d", k)
		}
	}

	if len(aCtx.scoredTests) != len(aCtx.tests) {
		// Try to deduce scoring remaining tests
		// zap.S().Info("Automatically inserting scores...")
		totalScore := 100
		for _, test := range aCtx.scoredTests {
			totalScore -= aCtx.tests[test].Score
		}

		// Since map order is ambiguous, get an ordered list of test IDs.
		// Regrettably, there is not easy way to do the set difference of the keys of the map and the scoredTests
		// so we'll do an O(N^2) operation for clarity's sake.
		testIDs := []int{}
		for id := range aCtx.tests {
			ok := true
			for _, scID := range aCtx.scoredTests {
				if id == scID {
					ok = false
					break
				}
			}
			if ok {
				testIDs = append(testIDs, id)
			}
		}
		sort.Ints(testIDs)

		n := len(aCtx.tests) - len(aCtx.scoredTests)
		perTest := totalScore/n + 1
		toSub := n - totalScore%n
		k := 0
		for _, i := range testIDs {
			if aCtx.tests[i].Score > 0 {
				continue
			}
			tst := aCtx.tests[i]
			tst.Score = perTest
			if k < toSub {
				tst.Score--
			}
			aCtx.tests[i] = tst
			k++
		}
	}

	// If we are loading an archive, the user might want to remove all tests first
	// So let's do it for them
	if err := base.OrphanTests(ctx, pb.ID); err != nil {
		zap.S().Warn(err)
		return err
	}

	createdTests := map[int]kilonova.Test{}

	for testID, v := range aCtx.tests {
		var test kilonova.Test
		test.ProblemID = pb.ID
		test.VisibleID = testID
		test.Score = v.Score
		if err := base.CreateTest(ctx, &test); err != nil {
			zap.S().Warn(err)
			return err
		}

		createdTests[testID] = test

		f, err := v.InFile.Open()
		if err != nil {
			return kilonova.WrapError(err, "Couldn't open() input file")
		}
		if err := base.SaveTestInput(test.ID, f); err != nil {
			zap.S().Warn("Couldn't create test input", err)
			f.Close()
			return kilonova.WrapError(err, "Couldn't create test input")
		}
		f.Close()
		f, err = v.OutFile.Open()
		if err != nil {
			return kilonova.WrapError(err, "Couldn't open() output file")
		}
		if err := base.SaveTestOutput(test.ID, f); err != nil {
			zap.S().Warn("Couldn't create test output", err)
			f.Close()
			return kilonova.WrapError(err, "Couldn't create test output")
		}
		f.Close()
	}

	if aCtx.props != nil {
		shouldUpd := false
		upd := kilonova.ProblemUpdate{}
		if aCtx.props.MemoryLimit != nil {
			shouldUpd = true
			upd.MemoryLimit = aCtx.props.MemoryLimit
		}
		if aCtx.props.TimeLimit != nil {
			shouldUpd = true
			upd.TimeLimit = aCtx.props.TimeLimit
		}
		if aCtx.props.DefaultPoints != nil {
			shouldUpd = true
			upd.DefaultPoints = aCtx.props.DefaultPoints
		}
		if aCtx.props.Source != nil {
			shouldUpd = true
			upd.SourceCredits = aCtx.props.Source
		}
		if aCtx.props.Author != nil {
			shouldUpd = true
			upd.AuthorCredits = aCtx.props.Author
		}
		if aCtx.props.ConsoleInput != nil {
			shouldUpd = true
			upd.ConsoleInput = aCtx.props.ConsoleInput
		}
		if aCtx.props.TestName != nil {
			shouldUpd = true
			upd.TestName = aCtx.props.TestName
		}

		if shouldUpd {
			if err := base.UpdateProblem(ctx, pb.ID, upd, nil); err != nil {
				zap.S().Warn(err)
				return kilonova.WrapError(err, "Couldn't update problem medatada")
			}
		}

		if aCtx.props.Subtasks != nil {
			if err := base.DeleteSubTasks(ctx, pb.ID); err != nil {
				zap.S().Warn(err)
				return kilonova.WrapError(err, "Couldn't delete existing subtasks")
			}
			for stkId, stk := range aCtx.props.Subtasks {
				outStk := kilonova.SubTask{
					ProblemID: pb.ID,
					VisibleID: stkId,
					Score:     stk.Score,
					Tests:     []int{},
				}
				for _, test := range stk.Tests {
					if tt, exists := createdTests[test]; !exists {
						return kilonova.Statusf(400, "Test %d not found in added tests. Aborting subtask creation", test)
					} else {
						outStk.Tests = append(outStk.Tests, tt.ID)
					}
				}

				if err := base.CreateSubTask(ctx, &outStk); err != nil {
					zap.S().Warn(err)
					return kilonova.WrapError(err, "Couldn't create subtask")
				}
			}
		}
	}

	return nil
}

// TODO: Add attachments when support for importing is added above.
func GenerateArchive(ctx context.Context, pb *kilonova.Problem, w io.Writer, base *sudoapi.BaseAPI) *kilonova.StatusError {
	ar := zip.NewWriter(w)
	tests, err := base.Tests(ctx, pb.ID)
	defer ar.Close()
	if err != nil {
		return err
	}

	testName := strings.TrimSpace(pb.TestName)
	if testName == "" {
		testName = slug.Make(testName)
	}

	// First, save the tests
	for _, test := range tests {
		{
			f, err := ar.Create(fmt.Sprintf("%d-%s.in", test.VisibleID, testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := base.TestInput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test input")
			}

			if _, err := io.Copy(f, r); err != nil {
				r.Close()
				return kilonova.WrapError(err, "Couldn't save test input file")
			}
			r.Close()
		}
		{
			f, err := ar.Create(fmt.Sprintf("%d-%s.ok", test.VisibleID, testName))
			if err != nil {
				return kilonova.WrapError(err, "Couldn't create archive file")
			}

			r, err := base.TestOutput(test.ID)
			if err != nil {
				return kilonova.WrapError(err, "Couldn't get test output")
			}

			if _, err := io.Copy(f, r); err != nil {
				r.Close()
				return kilonova.WrapError(err, "Couldn't save test output file")
			}
			r.Close()
		}
	}
	{
		// Then, the scores
		testFile, err := ar.Create("tests.txt")
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive tests.txt file")
		}
		for _, test := range tests {
			fmt.Fprintf(testFile, "%d %d\n", test.VisibleID, test.Score)
		}
	}
	{
		// Lastly, grader.properties
		subtasks, err1 := base.SubTasks(ctx, pb.ID)
		if err1 != nil {
			return err1
		}

		if len(subtasks) == 0 {
			// Just leave early if there are no subtasks
			return nil
		}

		gr, err := ar.Create("grader.properties")
		if err != nil {
			return kilonova.WrapError(err, "Couldn't create archive grader.properties file")
		}

		tmap := map[int]*kilonova.Test{}
		for _, test := range tests {
			tmap[test.ID] = test
		}

		groups := []string{}
		weights := []string{}

		for _, st := range subtasks {
			group := ""
			for i, t := range st.Tests {
				if i > 0 {
					group += ";"
				}
				tt, ok := tmap[t]
				if !ok {
					zap.S().Warn("Couldn't find test in test map")
				} else {
					group += strconv.Itoa(tt.VisibleID)
				}
			}
			groups = append(groups, group)
			weights = append(weights, strconv.Itoa(st.Score))
		}
		fmt.Fprintf(gr, "groups=%s\n", strings.Join(groups, ","))
		fmt.Fprintf(gr, "weights=%s\n", strings.Join(weights, ","))
		fmt.Fprintf(gr, "time=%f\n", pb.TimeLimit)
		fmt.Fprintf(gr, "memory=%f\n", float64(pb.MemoryLimit)/1024.0)
		if pb.DefaultPoints != 0 {
			fmt.Fprintf(gr, "default_score=%d\n", pb.DefaultPoints)
		}
		if pb.AuthorCredits != "" {
			fmt.Fprintf(gr, "author=%s\n", pb.AuthorCredits)
		}
		if pb.SourceCredits != "" {
			fmt.Fprintf(gr, "source=%s\n", pb.SourceCredits)
		}
		fmt.Fprintf(gr, "console_input=%t\n", pb.ConsoleInput)
		fmt.Fprintf(gr, "test_name=%s\n", testName)
	}
	return nil
}
