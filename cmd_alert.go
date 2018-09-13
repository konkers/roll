package roll

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Duration struct {
	time.Duration
}
type Time struct {
	time.Time
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var err error
	d.Duration, err = time.ParseDuration(v)
	if err != nil {
		return err
	}
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(time.RFC1123))
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	var err error
	t.Time, err = time.Parse(time.RFC1123, v)
	if err != nil {
		return err
	}
	return nil
}

type Alert struct {
	ID        int      `json:"id" storm:"id,increment"`
	Period    Duration `json:"period"`
	NextAlert Time     `json:"next_alert"`
	Message   string   `json:"message"`
}

type AlertService struct {
	bot *Bot
}

func (s *AlertService) tick(t time.Time) {
	var alerts []Alert
	err := s.bot.DB.From("alert").All(&alerts)
	if err != nil {
		log.Printf("Can't get alerts: %v", err)
		return
	}

	for _, alert := range alerts {
		if alert.NextAlert.Before(t) {
			log.Printf("Saying \"%s\"", alert.Message)
			s.bot.ircClient.Say(s.bot.Config.Channel, alert.Message)
			alert.NextAlert.Time = t.Add(alert.Period.Duration)
			s.bot.DB.From("alert").Save(&alert)
		}
	}
}

func (s *AlertService) worker() {
	for {
		ticker := time.NewTicker(1 * time.Minute)
		select {
		case t := <-ticker.C:
			s.tick(t)
		}
	}
}

func NewAlertService(bot *Bot) *AlertService {
	s := &AlertService{
		bot: bot,
	}
	go s.worker()
	return s
}

func (s *AlertService) New(r *http.Request, alert *Alert, id *int) error {
	alert.ID = 0
	return s.Update(r, alert, id)
}

func (s *AlertService) Update(r *http.Request, alert *Alert, id *int) error {
	if !s.bot.isAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.bot.DB.From("alert").Save(alert)
	if err != nil {
		*id = -1
		return err
	}

	*id = alert.ID
	return nil
}

func (s *AlertService) Get(r *http.Request, id *int, alert *Alert) error {
	return s.bot.DB.From("alert").One("ID", *id, alert)
}

func (s *AlertService) Trigger(r *http.Request, id *int, resp *int) error {
	var alert Alert
	err := s.bot.DB.From("alert").One("ID", *id, alert)
	if err != nil {
		return err
	}
	alert.NextAlert.Time = time.Now()
	return s.bot.DB.From("alert").Save(alert)
}
