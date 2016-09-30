package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
)

type EvalContext struct {
	Firing          bool
	IsTestRun       bool
	EvalMatches     []*EvalMatch
	Logs            []*ResultLogEntry
	Error           error
	Description     string
	StartTime       time.Time
	EndTime         time.Time
	Rule            *Rule
	log             log.Logger
	dashboardSlug   string
	ImagePublicUrl  string
	ImageOnDiskPath string
	NoDataFound     bool
	RetryCount      int

	Cancel  context.CancelFunc
	Context context.Context
}

func (evalContext *EvalContext) Deadline() (deadline time.Time, ok bool) {
	return evalContext.Deadline()
}

func (evalContext *EvalContext) Done() <-chan struct{} {
	return evalContext.Context.Done()
}

func (evalContext *EvalContext) Err() error {
	return evalContext.Context.Err()
}

func (evalContext *EvalContext) Value(key interface{}) interface{} {
	return evalContext.Context.Value(key)
}

type StateDescription struct {
	Color string
	Text  string
	Data  string
}

func (c *EvalContext) GetStateModel() *StateDescription {
	switch c.Rule.State {
	case m.AlertStateOK:
		return &StateDescription{
			Color: "#36a64f",
			Text:  "OK",
		}
	case m.AlertStateNoData:
		return &StateDescription{
			Color: "#888888",
			Text:  "No Data",
		}
	case m.AlertStateExecError:
		return &StateDescription{
			Color: "#000",
			Text:  "Execution Error",
		}
	case m.AlertStateAlerting:
		return &StateDescription{
			Color: "#D63232",
			Text:  "Alerting",
		}
	default:
		panic("Unknown rule state " + c.Rule.State)
	}
}

func (a *EvalContext) GetDurationMs() float64 {
	return float64(a.EndTime.Nanosecond()-a.StartTime.Nanosecond()) / float64(1000000)
}

func (c *EvalContext) GetNotificationTitle() string {
	return "[" + c.GetStateModel().Text + "] " + c.Rule.Name
}

func (c *EvalContext) GetDashboardSlug() (string, error) {
	if c.dashboardSlug != "" {
		return c.dashboardSlug, nil
	}

	slugQuery := &m.GetDashboardSlugByIdQuery{Id: c.Rule.DashboardId}
	if err := bus.Dispatch(slugQuery); err != nil {
		return "", err
	}

	c.dashboardSlug = slugQuery.Result
	return c.dashboardSlug, nil
}

func (c *EvalContext) GetRuleUrl() (string, error) {
	if slug, err := c.GetDashboardSlug(); err != nil {
		return "", err
	} else {
		ruleUrl := fmt.Sprintf("%sdashboard/db/%s?fullscreen&edit&tab=alert&panelId=%d", setting.AppUrl, slug, c.Rule.PanelId)
		return ruleUrl, nil
	}
}

func NewEvalContext(grafanaCtx context.Context, rule *Rule) *EvalContext {
	timeout := time.Duration(time.Second * 20)
	ctx, cancelFn := context.WithTimeout(grafanaCtx, timeout)

	return &EvalContext{
		Cancel:      cancelFn,
		Context:     ctx,
		StartTime:   time.Now(),
		Rule:        rule,
		Logs:        make([]*ResultLogEntry, 0),
		EvalMatches: make([]*EvalMatch, 0),
		log:         log.New("alerting.evalContext"),
		RetryCount:  0,
	}
}
