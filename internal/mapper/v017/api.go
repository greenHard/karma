package v017

import (
	"net/http"
	"net/url"
	"path"
	"sort"
	"time"

	httptransport "github.com/go-openapi/runtime/client"

	"github.com/prymitive/karma/internal/mapper"
	"github.com/prymitive/karma/internal/mapper/v017/client"
	"github.com/prymitive/karma/internal/mapper/v017/client/alertgroup"
	"github.com/prymitive/karma/internal/mapper/v017/client/general"
	"github.com/prymitive/karma/internal/mapper/v017/client/silence"
	"github.com/prymitive/karma/internal/models"
)

func newClient(uri string, headers map[string]string, httpTransport http.RoundTripper) *client.Alertmanager {
	u, _ := url.Parse(uri)

	transport := httptransport.New(u.Host, path.Join(u.Path, "/api/v2"), []string{u.Scheme})

	if httpTransport != nil {
		transport.Transport = mapper.SetHeaders(httpTransport, headers)
	} else {
		transport.Transport = mapper.SetHeaders(transport.Transport, headers)
	}

	if u.User.Username() != "" {
		username := u.User.Username()
		password, _ := u.User.Password()
		transport.Transport = mapper.SetAuth(transport.Transport, username, password)
	}

	c := client.New(transport, nil)
	return c
}

// Alerts will fetch all alert groups from the API
func groups(c *client.Alertmanager, timeout time.Duration) ([]models.AlertGroup, error) {
	ret := []models.AlertGroup{}

	groups, err := c.Alertgroup.GetAlertGroups(alertgroup.NewGetAlertGroupsParamsWithTimeout(timeout))
	if err != nil {
		return []models.AlertGroup{}, err
	}
	for _, group := range groups.Payload {
		g := models.AlertGroup{
			Receiver: *group.Receiver.Name,
			Labels:   group.Labels,
		}
		for _, alert := range group.Alerts {
			a := models.Alert{
				Receiver:     *group.Receiver.Name,
				Annotations:  models.AnnotationsFromMap(alert.Annotations),
				Labels:       alert.Labels,
				StartsAt:     time.Time(*alert.StartsAt),
				GeneratorURL: alert.GeneratorURL.String(),
				State:        *alert.Status.State,
				InhibitedBy:  alert.Status.InhibitedBy,
				SilencedBy:   alert.Status.SilencedBy,
			}
			sort.Strings(a.InhibitedBy)
			sort.Strings(a.SilencedBy)
			a.UpdateFingerprints()
			g.Alerts = append(g.Alerts, a)
		}
		ret = append(ret, g)
	}

	return ret, nil
}

func silences(c *client.Alertmanager, timeout time.Duration) ([]models.Silence, error) {
	ret := []models.Silence{}

	silences, err := c.Silence.GetSilences(silence.NewGetSilencesParamsWithTimeout(timeout))
	if err != nil {
		return ret, err
	}

	for _, s := range silences.Payload {
		us := models.Silence{
			ID:        *s.ID,
			StartsAt:  time.Time(*s.StartsAt),
			EndsAt:    time.Time(*s.EndsAt),
			CreatedBy: *s.CreatedBy,
			Comment:   *s.Comment,
		}
		for _, m := range s.Matchers {
			sm := models.SilenceMatcher{
				Name:    *m.Name,
				Value:   *m.Value,
				IsRegex: *m.IsRegex,
			}
			us.Matchers = append(us.Matchers, sm)
		}
		ret = append(ret, us)
	}

	return ret, nil
}

func status(c *client.Alertmanager, timeout time.Duration) (models.AlertmanagerStatus, error) {
	ret := models.AlertmanagerStatus{}

	status, err := c.General.GetStatus(general.NewGetStatusParamsWithTimeout(timeout))
	if err != nil {
		return ret, err
	}

	ret.Version = *status.Payload.VersionInfo.Version
	ret.ID = status.Payload.Cluster.Name
	for _, p := range status.Payload.Cluster.Peers {
		ret.PeerIDs = append(ret.PeerIDs, *p.Name)
	}

	return ret, nil
}
