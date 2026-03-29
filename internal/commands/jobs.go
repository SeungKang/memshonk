package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/jobsctl"
)

const (
	JobsCommandName = "jobs"
)

func NewJobsCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &JobsCommand{
		jobs: config.Session.Jobs(),
	}

	root := fx.NewCommand(JobsCommandName, "manage background jobs", cmd.ls)

	root.AddSubcommand("ls", "list jobs", cmd.ls)

	rm := root.AddSubcommand("rm", "stop a job by its job ID", cmd.rm)
	rm.FlagSet.StringNf(&cmd.targetJobId, fx.ArgConfig{
		Name:        "job-id",
		Description: "The job ID",
		Required:    true,
	})

	return root
}

type JobsCommand struct {
	jobs *jobsctl.Ctl

	targetJobId string
}

func (o *JobsCommand) ls(ctx context.Context) (fx.CommandResult, error) {
	jobs := o.jobs.List()

	sort.SliceStable(jobs, func(i int, j int) bool {
		return jobs[i].ID() < jobs[j].ID()
	})

	sb := strings.Builder{}

	for i, job := range jobs {
		info := job.Info()

		if info.RegisterConfig.Argv[0] == JobsCommandName {
			// Skip this current job.
			continue
		}

		// "memshonk.0123456"
		sb.WriteString(fmt.Sprintf("%-16s", info.ID))

		sb.WriteByte('"')
		sb.WriteString(strings.Join(info.RegisterConfig.Argv, " "))
		sb.WriteByte('"')

		if info.HasPID {
			sb.WriteString(" (pid: ")
			sb.WriteString(strconv.FormatInt(int64(info.PID), 10))
			sb.WriteString(")")
		}

		sb.WriteString(" ")
		sb.WriteString(info.StartedAt.Format(time.Stamp))

		if i > 0 && i != len(jobs)-1 {
			sb.WriteByte('\n')
		}
	}

	if sb.Len() == 0 {
		return nil, nil
	}

	return fx.NewHumanCommandResult(sb.String()), nil
}

func (o *JobsCommand) rm(ctx context.Context) (fx.CommandResult, error) {
	job, err := o.jobs.Lookup(o.targetJobId)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup job: %q - %w",
			o.targetJobId, err)
	}

	job.CancelSync(ctx)

	return nil, nil
}
