package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/santifer/career-ops/dashboard/internal/data"
	"github.com/santifer/career-ops/dashboard/internal/model"
	"github.com/santifer/career-ops/dashboard/internal/theme"
	"github.com/santifer/career-ops/dashboard/internal/ui/screens"
)

type viewState int

const (
	viewPipeline viewState = iota
	viewReport
	viewProgress
	viewOutput
	viewTasks
)

type appModel struct {
	pipeline        screens.PipelineModel
	viewer          screens.ViewerModel
	progress        screens.ProgressModel
	output          screens.OutputModel
	tasks           screens.TasksModel
	taskEntries     []*screens.TaskEntry
	nextTaskID      int
	activeTaskID    int // task being viewed in output view, -1 if none
	state           viewState
	careerOpsPath   string
	theme           theme.Theme
	progressMetrics model.ProgressMetrics
}

func (m appModel) Init() tea.Cmd {
	return nil
}

func (m *appModel) findTask(id int) *screens.TaskEntry {
	for _, t := range m.taskEntries {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (m *appModel) runningCount() int {
	count := 0
	for _, t := range m.taskEntries {
		if t.Status == "running" {
			count++
		}
	}
	return count
}

func (m *appModel) reloadPipeline() {
	apps := data.ParseApplications(m.careerOpsPath)
	pending := data.ParsePendingOffers(m.careerOpsPath)
	metrics := data.ComputeMetrics(apps)
	m.progressMetrics = data.ComputeProgressMetrics(apps)
	old := m.pipeline
	m.pipeline = screens.NewPipelineModel(
		m.theme,
		apps, pending, metrics, m.careerOpsPath,
		old.Width(), old.Height(),
	)
	m.pipeline.CopyReportCache(&old)
	m.pipeline.CopyViewState(&old)
	m.pipeline.SetRunningTasks(m.runningCount())
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.pipeline.Resize(msg.Width, msg.Height)
		if m.state == viewReport {
			m.viewer.Resize(msg.Width, msg.Height)
		}
		if m.state == viewProgress {
			m.progress.Resize(msg.Width, msg.Height)
		}
		if m.state == viewOutput {
			m.output.Resize(msg.Width, msg.Height)
		}
		if m.state == viewTasks {
			m.tasks.Resize(msg.Width, msg.Height)
		}
		pm, cmd := m.pipeline.Update(msg)
		m.pipeline = pm
		return m, cmd

	case screens.PipelineClosedMsg:
		return m, tea.Quit

	case screens.PipelineLoadReportMsg:
		archetype, tldr, remote, comp := data.LoadReportSummary(msg.CareerOpsPath, msg.ReportPath)
		m.pipeline.EnrichReport(msg.ReportPath, archetype, tldr, remote, comp)
		return m, nil

	case screens.PipelineRefreshMsg:
		m.reloadPipeline()
		return m, nil

	case screens.PipelineBatchEvalMsg:
		evalAction := screens.ActionItem{Label: "Evaluate", Category: "claude", Command: "evaluate", NeedsApp: true}
		var cmds []tea.Cmd
		for _, app := range msg.Apps {
			taskID := m.nextTaskID
			m.nextTaskID++
			appCopy := app
			task := &screens.TaskEntry{
				ID:        taskID,
				Label:     fmt.Sprintf("Evaluate: %s — %s", appCopy.Company, appCopy.Role),
				Status:    "running",
				StartTime: time.Now(),
				JobURL:    appCopy.JobURL,
			}
			m.taskEntries = append(m.taskEntries, task)
			m.pipeline.SetPendingStatus(appCopy.JobURL, "Evaluating")
			runMsg := screens.PipelineRunActionMsg{Action: evalAction, App: &appCopy, CareerOpsPath: msg.CareerOpsPath}
			runner, cmd := screens.StartAction(runMsg, m.careerOpsPath, taskID)
			task.Runner = runner
			cmds = append(cmds, cmd)
		}
		m.pipeline.SetRunningTasks(m.runningCount())
		return m, tea.Batch(cmds...)

	case screens.PipelineUpdateStatusMsg:
		err := data.UpdateApplicationStatus(msg.CareerOpsPath, msg.App, msg.NewStatus)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: status update failed: %v\n", err)
		}
		m.reloadPipeline()
		return m, nil

	case screens.PipelineOpenReportMsg:
		m.viewer = screens.NewViewerModel(
			m.theme,
			msg.Path, msg.Title,
			m.pipeline.Width(), m.pipeline.Height(),
		)
		m.state = viewReport
		return m, nil

	case screens.ViewerClosedMsg:
		m.state = viewPipeline
		return m, nil

	case screens.PipelineOpenProgressMsg:
		m.progress = screens.NewProgressModel(
			m.theme,
			m.progressMetrics,
			m.pipeline.Width(), m.pipeline.Height(),
		)
		m.state = viewProgress
		return m, nil

	case screens.ProgressClosedMsg:
		m.state = viewPipeline
		return m, nil

	case screens.PipelineRunActionMsg:
		// Create a new task entry
		taskID := m.nextTaskID
		m.nextTaskID++
		task := &screens.TaskEntry{
			ID:        taskID,
			Label:     msg.Action.Label,
			Status:    "running",
			StartTime: time.Now(),
		}
		// Track JobURL for evaluate actions on pending offers
		if msg.Action.Command == "evaluate" && msg.App != nil && msg.App.JobURL != "" {
			task.JobURL = msg.App.JobURL
			m.pipeline.SetPendingStatus(msg.App.JobURL, "Evaluating")
		}
		m.taskEntries = append(m.taskEntries, task)

		// Start the subprocess
		runner, cmd := screens.StartAction(msg, m.careerOpsPath, taskID)
		task.Runner = runner

		// Update running count in pipeline
		m.pipeline.SetRunningTasks(m.runningCount())

		// Stay on pipeline — task runs in background
		return m, cmd

	case screens.ActionOutputMsg:
		task := m.findTask(msg.TaskID)
		if task == nil {
			return m, nil
		}
		if msg.Done {
			// Don't overwrite cancelled status — the process was killed intentionally
			if task.Status != "cancelled" {
				if msg.Error != nil {
					task.Status = "failed"
					task.Err = msg.Error
				} else if msg.ExitCode != 0 {
					task.Status = "failed"
					task.ExitCode = msg.ExitCode
				} else {
					task.Status = "completed"
				}
			}
			// Update pending offer status for evaluate tasks
			if task.JobURL != "" {
				switch task.Status {
				case "completed":
					m.pipeline.SetPendingStatus(task.JobURL, "Done")
				case "failed":
					m.pipeline.SetPendingStatus(task.JobURL, "Failed")
				case "cancelled":
					m.pipeline.SetPendingStatus(task.JobURL, "Cancelled")
				}
			}
			m.pipeline.SetRunningTasks(m.runningCount())
			// If we're viewing this task's output, update it
			if m.state == viewOutput && m.activeTaskID == msg.TaskID {
				m.output.AppendOutput(msg)
			}
			// If viewing tasks list, refresh it
			if m.state == viewTasks {
				m.tasks.UpdateTasks(m.taskEntries)
			}
			// Reload pipeline data in case the action modified files
			m.reloadPipeline()
			return m, nil
		}
		task.Lines = append(task.Lines, msg.Line)
		// If we're viewing this task's output, stream to it
		if m.state == viewOutput && m.activeTaskID == msg.TaskID {
			m.output.AppendOutput(msg)
		}
		return m, task.Runner.WaitForOutput(msg.TaskID)

	case screens.PipelineOpenTasksMsg:
		m.tasks = screens.NewTasksModel(
			m.theme,
			m.taskEntries,
			m.pipeline.Width(), m.pipeline.Height(),
		)
		m.state = viewTasks
		return m, nil

	case screens.TasksClosedMsg:
		m.state = viewPipeline
		m.reloadPipeline()
		return m, nil

	case screens.TasksViewOutputMsg:
		task := m.findTask(msg.TaskID)
		if task == nil {
			return m, nil
		}
		m.activeTaskID = msg.TaskID
		m.output = screens.NewOutputModel(
			m.theme,
			task.Label,
			m.pipeline.Width(), m.pipeline.Height(),
		)
		// Load existing output lines
		m.output.LoadLines(task.Lines, task.Status != "running")
		if task.Status != "running" {
			m.output.MarkDone(task.ExitCode, task.Err)
		}
		m.state = viewOutput
		return m, nil

	case screens.TasksCancelMsg:
		task := m.findTask(msg.TaskID)
		if task != nil && task.Status == "running" {
			task.Runner.Cancel()
			task.Status = "cancelled"
			if task.JobURL != "" {
				m.pipeline.SetPendingStatus(task.JobURL, "Cancelled")
			}
			m.pipeline.SetRunningTasks(m.runningCount())
			if m.state == viewTasks {
				m.tasks.UpdateTasks(m.taskEntries)
			}
		}
		return m, nil

	case screens.OutputClosedMsg:
		m.activeTaskID = -1
		// Return to tasks view if we came from there, otherwise pipeline
		if len(m.taskEntries) > 0 {
			m.tasks = screens.NewTasksModel(
				m.theme,
				m.taskEntries,
				m.pipeline.Width(), m.pipeline.Height(),
			)
			m.state = viewTasks
		} else {
			m.state = viewPipeline
		}
		return m, nil

	case screens.PipelineOpenURLMsg:
		url := msg.URL
		return m, func() tea.Msg {
			var cmd *exec.Cmd
			switch runtime.GOOS {
			case "darwin":
				cmd = exec.Command("open", url)
			case "linux":
				cmd = exec.Command("xdg-open", url)
			case "windows":
				cmd = exec.Command("cmd", "/c", "start", "", url)
			default:
				cmd = exec.Command("xdg-open", url)
			}
			_ = cmd.Run()
			return nil
		}

	default:
		switch m.state {
		case viewOutput:
			om, cmd := m.output.Update(msg)
			m.output = om
			return m, cmd
		case viewReport:
			vm, cmd := m.viewer.Update(msg)
			m.viewer = vm
			return m, cmd
		case viewTasks:
			tm, cmd := m.tasks.Update(msg)
			m.tasks = tm
			return m, cmd
		case viewProgress:
			pg, cmd := m.progress.Update(msg)
			m.progress = pg
			return m, cmd
		default:
			pm, cmd := m.pipeline.Update(msg)
			m.pipeline = pm
			return m, cmd
		}
	}
}

func (m appModel) View() string {
	switch m.state {
	case viewReport:
		return m.viewer.View()
	case viewProgress:
		return m.progress.View()
	case viewOutput:
		return m.output.View()
	case viewTasks:
		return m.tasks.View()
	default:
		return m.pipeline.View()
	}
}

func findCareerOpsRoot(start string) string {
	dir, _ := filepath.Abs(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "data", "applications.md")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "applications.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func main() {
	pathFlag := flag.String("path", "", "Path to career-ops directory")
	flag.Parse()

	careerOpsPath := *pathFlag
	if careerOpsPath == "" {
		// Walk up from cwd to find the project root
		cwd, _ := os.Getwd()
		careerOpsPath = findCareerOpsRoot(cwd)
		if careerOpsPath == "" {
			fmt.Fprintf(os.Stderr, "Error: could not find applications.md in any parent directory. Use --path to specify the career-ops root.\n")
			os.Exit(1)
		}
	}

	// Load applications
	apps := data.ParseApplications(careerOpsPath)
	if apps == nil {
		fmt.Fprintf(os.Stderr, "Error: could not find applications.md in %s or %s/data/\n", careerOpsPath, careerOpsPath)
		os.Exit(1)
	}

	// Load pending offers from pipeline.md
	pending := data.ParsePendingOffers(careerOpsPath)

	// Compute metrics
	metrics := data.ComputeMetrics(apps)
	progressMetrics := data.ComputeProgressMetrics(apps)

	// Batch-load all report summaries
	t := theme.NewTheme("auto")
	pm := screens.NewPipelineModel(t, apps, pending, metrics, careerOpsPath, 120, 40)

	for _, app := range apps {
		if app.ReportPath == "" {
			continue
		}
		archetype, tldr, remote, comp := data.LoadReportSummary(careerOpsPath, app.ReportPath)
		if archetype != "" || tldr != "" || remote != "" || comp != "" {
			pm.EnrichReport(app.ReportPath, archetype, tldr, remote, comp)
		}
	}

	m := appModel{
		pipeline:        pm,
		careerOpsPath:   careerOpsPath,
		theme:           t,
		progressMetrics: progressMetrics,
		activeTaskID:    -1,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
