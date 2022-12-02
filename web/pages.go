package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"reflect"
	"strconv"
	tparse "text/template/parse"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"go.uber.org/zap"
)

type EditTopbar struct {
	Page   string
	PageID int
}

type ReqContext struct {
	User     *kilonova.UserFull
	Language string
}

type ProblemParams struct {
	Ctx           *ReqContext
	ProblemEditor bool

	Problem     *kilonova.Problem
	Attachments []*kilonova.Attachment

	Markdown  template.HTML
	Languages map[string]eval.Language
}

type ProblemListParams struct {
	Ctx         *ReqContext
	ProblemList *kilonova.ProblemList
}

type SubTaskEditParams struct {
	Ctx     *ReqContext
	Problem *kilonova.Problem
	SubTask *kilonova.SubTask
	Topbar  *EditTopbar

	ctx  context.Context
	base *sudoapi.BaseAPI
}

func (s *SubTaskEditParams) TestSubTasks(id int) string {
	sts, err := s.base.SubTasksByTest(s.ctx, s.Problem.ID, id)
	if err != nil || sts == nil || len(sts) == 0 {
		return "-"
	}
	out := strconv.Itoa(sts[0].VisibleID)
	for id, st := range sts {
		if id > 0 {
			out += fmt.Sprintf(", %d", st.VisibleID)
		}
	}
	return out
}

func (s *SubTaskEditParams) TestInSubTask(test *kilonova.Test) bool {
	for _, id := range s.SubTask.Tests {
		if id == test.ID {
			return true
		}
	}
	return false
}

type TestEditParams struct {
	Ctx     *ReqContext
	Problem *kilonova.Problem
	Test    *kilonova.Test
	Topbar  *EditTopbar

	base *sudoapi.BaseAPI
}

type testDataType struct {
	In  string
	Out string

	OkIn  bool
	OkOut bool
}

const readLimit = 1024 * 1024 // 1MB

func ReadOrTruncate(r io.Reader) ([]byte, bool) {
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, r, readLimit); err != nil {
		if errors.Is(err, io.EOF) {
			return buf.Bytes(), true
		}
		zap.S().Warn(err)
		return []byte("err"), false
	}

	return []byte("Files larger than 1MB cannot be displayed"), false
}

func (t *TestEditParams) GetFullTests() testDataType {
	in, err := t.base.TestInput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer in.Close()

	out, err := t.base.TestOutput(t.Test.ID)
	if err != nil {
		return testDataType{In: "err", Out: "err"}
	}
	defer out.Close()

	inData, okIn := ReadOrTruncate(in)
	outData, okOut := ReadOrTruncate(out)

	return testDataType{
		In:   string(inData),
		OkIn: okIn,

		Out:   string(outData),
		OkOut: okOut,
	}
}

type IndexParams struct {
	Ctx *ReqContext

	Version     string
	Description string
}

type ProblemListingParams struct {
	User      *kilonova.UserBrief
	Problems  []*kilonova.Problem
	ShowScore bool
	MultiCols bool
}

type PblistParams struct {
	User   *kilonova.UserBrief
	Ctx    *ReqContext
	Pblist *kilonova.ProblemList
	Open   bool
}

type ProfileParams struct {
	Ctx *ReqContext

	ContentUser  *kilonova.UserFull
	UserProblems []*kilonova.Problem
}

type AuditLogParams struct {
	Ctx *ReqContext

	Logs     []*kilonova.AuditLog
	NumPages int
}

type StatusParams struct {
	Ctx *ReqContext

	Code    int
	Message string
}

type MarkdownParams struct {
	Ctx      *ReqContext
	Markdown template.HTML
	Title    string
}

type SimpleParams struct {
	Ctx *ReqContext
}

func GenContext(r *http.Request) *ReqContext {
	return &ReqContext{
		User:     util.UserFull(r),
		Language: util.Language(r),
	}
}

type VerifiedEmailParams struct {
	Ctx *ReqContext

	ContentUser *kilonova.UserBrief
}

type PasswordResetParams struct {
	Ctx *ReqContext

	User      *kilonova.UserFull
	RequestID string
}

type SubParams struct {
	Ctx        *ReqContext
	Submission *kilonova.Submission
}

type PasteParams struct {
	Ctx   *ReqContext
	Paste *kilonova.SubmissionPaste
}

func doWalk(filename string, nodes ...tparse.Node) bool {
	ok := true
	for _, node := range nodes {
		tp := reflect.Indirect(reflect.ValueOf(node))
		if val := tp.FieldByName("List"); val.IsValid() {
			if val.Kind() == reflect.Pointer {
				val = reflect.Indirect(val)
			}
			if nodes := val.FieldByName("Nodes"); nodes.IsValid() {
				if nodes.Kind() != reflect.Slice {
					panic("Wtf")
				}
				ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
			}
		}
		if nodes := tp.FieldByName("Nodes"); nodes.IsValid() {
			if nodes.Kind() != reflect.Slice {
				panic("Wtf")
			}
			ok = ok && doWalk(filename, nodes.Interface().([]tparse.Node)...)
		}
		//spew.Dump(node.Type(), node.Position(), node.String())
		if rnode, ok := node.(*tparse.ActionNode); ok {
			for _, cmd := range rnode.Pipe.Cmds {
				if len(cmd.Args) == 0 {
					continue
				}
				val, ok := cmd.Args[0].(*tparse.IdentifierNode)
				if !ok || val.Ident != "getText" || len(cmd.Args) < 2 {
					continue
				}
				key := cmd.Args[1].(*tparse.StringNode).Text
				if !kilonova.TranslationKeyExists(key) {
					zap.S().Infof("Template static analysis failed: Unknown translation key %q in file %s", key, filename)
					ok = false
				}
			}
			//spew.Dump(rnode)
		}
	}
	return ok
}

func parse(optFuncs template.FuncMap, files ...string) *template.Template {
	templs, err := fs.Sub(templateDir, "templ")
	if err != nil {
		zap.S().Fatal(err)
	}
	t := template.New("layout.html")
	if optFuncs != nil {
		t = t.Funcs(optFuncs)
	}
	files = append(files, "util/navbar.html", "util/footer.html")
	if true { //config.Common.Debug { // && false {
		f, err := fs.ReadFile(templs, files[0])
		if err != nil {
			zap.S().Fatal(err)
		}
		ptrees, err := tparse.Parse(files[0], string(f), "{{", "}}", optFuncs, builtinTemporaryTemplate())
		if err != nil {
			zap.S().Fatal(err)
		}
		tree := ptrees["content"]
		doWalk(files[0], tree.Root)
	}
	return template.Must(t.ParseFS(templs, append([]string{"layout.html"}, files...)...))
}

func builtinTemporaryTemplate() template.FuncMap {
	names := []string{"and", "call", "html", "index", "slice", "js", "len", "not", "or", "print", "printf", "println", "urlquery", "eq", "ge", "gt", "le", "lt", "ne"}
	rez := make(template.FuncMap)
	for _, name := range names {
		rez[name] = func() {}
	}
	return rez
}
