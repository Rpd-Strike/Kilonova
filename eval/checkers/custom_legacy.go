package checkers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
)

var _ eval.Task = &legacyCustomCheckerTask{}

type legacyCustomCheckerTask struct {
	c    *CustomChecker
	pOut io.Reader
	cIn  io.Reader
	cOut io.Reader

	// filled by Execute
	score  int
	output string
}

func (job *legacyCustomCheckerTask) Execute(ctx context.Context, box eval.Sandbox) error {
	lang, ok := eval.Langs[eval.GetLangByFilename(job.c.filename)]
	if !ok {
		job.output = ErrOut
		return nil
	}

	if err := box.WriteFile("/box/program.out", job.pOut, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := box.WriteFile("/box/correct.in", job.cIn, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := box.WriteFile("/box/correct.out", job.cOut, 0644); err != nil {
		job.output = ErrOut
		return nil
	}
	if err := eval.CopyInBox(box, path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", -job.c.sub.ID)), lang.CompiledName); err != nil {
		job.output = ErrOut
		return nil
	}

	goodCmd, err := eval.MakeGoodCommand(lang.RunCommand)
	if err != nil {
		job.output = ErrOut
		return nil
	}
	goodCmd = append(goodCmd, "/box/program.out", "/box/correct.out", "/box/correct.in")

	var out bytes.Buffer

	conf := &eval.RunConfig{
		Stdout: &out,

		MemoryLimit: 512 * 1024,

		WallTimeLimit: 20,

		MaxProcs: 2,
	}

	if _, err := box.RunCommand(ctx, goodCmd, conf); err != nil {
		job.output = ErrOut
		return nil
	}

	if _, err := fmt.Fscanf(&out, "%d ", &job.score); err != nil {
		job.output = "Wrong checker output"
		return nil
	}

	job.output = strings.TrimSpace(out.String())
	return nil
}
