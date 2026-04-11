package screens

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/santifer/career-ops/dashboard/internal/model"
)

// PipelineRunActionMsg is emitted when the user selects an action from the menu.
type PipelineRunActionMsg struct {
	Action        ActionItem
	App           *model.CareerApplication
	CareerOpsPath string
}

// ActionOutputMsg carries a chunk of streaming output from a subprocess.
type ActionOutputMsg struct {
	TaskID   int
	Line     string
	Done     bool
	ExitCode int
	Error    error
}

// ActionRunner manages a running subprocess.
type ActionRunner struct {
	outputChan chan string
	exitCode   int
	err        error
	cancel     context.CancelFunc
}

// Cancel stops the running subprocess.
func (r *ActionRunner) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}

// WaitForOutput returns a tea.Cmd that blocks until the next line of output.
func (r *ActionRunner) WaitForOutput(taskID int) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-r.outputChan
		if !ok {
			return ActionOutputMsg{TaskID: taskID, Done: true, ExitCode: r.exitCode, Error: r.err}
		}
		return ActionOutputMsg{TaskID: taskID, Line: line}
	}
}

// StartAction spawns a subprocess for the given action and returns a runner + initial cmd.
func StartAction(msg PipelineRunActionMsg, careerOpsPath string, taskID int) (*ActionRunner, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())

	var cmd *exec.Cmd

	if msg.Action.Category == "claude" {
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			ch := make(chan string)
			close(ch)
			r := &ActionRunner{
				outputChan: ch,
				err:        fmt.Errorf("claude CLI not found — install from https://docs.anthropic.com/en/docs/claude-code"),
				cancel:     cancel,
			}
			return r, r.WaitForOutput(taskID)
		}
		prompt := buildClaudePrompt(msg.Action, msg.App)
		cmd = exec.CommandContext(ctx, claudePath, "-p", prompt)
		cmd.Dir = careerOpsPath
	} else {
		nodePath, err := exec.LookPath("node")
		if err != nil {
			ch := make(chan string)
			close(ch)
			r := &ActionRunner{
				outputChan: ch,
				err:        fmt.Errorf("node not found — install Node.js"),
				cancel:     cancel,
			}
			return r, r.WaitForOutput(taskID)
		}
		cmd = exec.CommandContext(ctx, nodePath, msg.Action.Command)
		cmd.Dir = careerOpsPath
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch := make(chan string)
		close(ch)
		r := &ActionRunner{outputChan: ch, err: err, cancel: cancel}
		return r, r.WaitForOutput(taskID)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		ch := make(chan string)
		close(ch)
		r := &ActionRunner{outputChan: ch, err: err, cancel: cancel}
		return r, r.WaitForOutput(taskID)
	}

	ch := make(chan string, 100)
	runner := &ActionRunner{outputChan: ch, cancel: cancel}

	if err := cmd.Start(); err != nil {
		close(ch)
		runner.err = err
		return runner, runner.WaitForOutput(taskID)
	}

	// Read stdout and stderr concurrently — MultiReader would block on
	// stdout until EOF before reading stderr, causing 0 output for
	// processes like `claude -p` that stream to stderr.
	var wg sync.WaitGroup
	scanPipe := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
	}
	wg.Add(2)
	go scanPipe(stdout)
	go scanPipe(stderr)

	go func() {
		wg.Wait()
		_ = cmd.Wait()
		if cmd.ProcessState != nil {
			runner.exitCode = cmd.ProcessState.ExitCode()
		}
		close(ch)
	}()

	return runner, runner.WaitForOutput(taskID)
}

func buildClaudePrompt(action ActionItem, app *model.CareerApplication) string {
	switch action.Command {
	case "evaluate":
		if app != nil && app.JobURL != "" {
			return fmt.Sprintf("/career-ops evaluate %s", app.JobURL)
		}
		if app != nil {
			return fmt.Sprintf("/career-ops evaluate for %s - %s", app.Company, app.Role)
		}
		return "/career-ops evaluate"

	case "pdf":
		if app != nil {
			return fmt.Sprintf("/career-ops pdf for %s - %s", app.Company, app.Role)
		}
		return "/career-ops pdf"

	case "contact":
		if app != nil {
			return fmt.Sprintf("/career-ops contact for %s - %s", app.Company, app.Role)
		}
		return "/career-ops contact"

	case "deep":
		if app != nil {
			return fmt.Sprintf("/career-ops deep %s", app.Company)
		}
		return "/career-ops deep"

	case "interview-prep":
		if app != nil {
			return fmt.Sprintf("/career-ops interview-prep for %s - %s", app.Company, app.Role)
		}
		return "/career-ops interview-prep"

	case "compare":
		return "/career-ops compare"

	case "followup":
		return "/career-ops followup"

	case "scan":
		return "/career-ops scan"

	default:
		return "/career-ops " + action.Command
	}
}
