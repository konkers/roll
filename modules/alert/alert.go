package alert

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asdine/storm"
	"github.com/konkers/roll"
)

type Alert struct {
	ID        int           `json:"id" storm:"id,increment"`
	Period    roll.Duration `json:"period"`
	NextAlert roll.Time     `json:"next_alert"`
	Message   string        `json:"message"`
}

type AlertModule struct {
	bot *roll.Bot
	db  storm.Node

	service *AlertService
	closeC  chan struct{}
}

type AlertService struct {
	module *AlertModule
}

func init() {
	roll.RegisterModuleFactory(NewAlertModule, "alert")
}

func NewAlertModule(bot *roll.Bot, dbBucket storm.Node) (roll.Module, error) {
	module := &AlertModule{
		bot:    bot,
		db:     dbBucket,
		closeC: make(chan struct{}),
	}

	module.service = NewAlertService(module)

	return module, nil
}

func NewAlertService(module *AlertModule) *AlertService {
	s := &AlertService{
		module: module,
	}
	return s
}

func (m *AlertModule) Start() error {
	go m.worker()
	return nil
}

func (m *AlertModule) Stop() error {
	close(m.closeC)
	return nil
}

func (m *AlertModule) tick(t time.Time) {
	var alerts []Alert
	err := m.db.All(&alerts)
	if err != nil {
		log.Printf("Can't get alerts: %v", err)
		return
	}

	for _, alert := range alerts {
		if alert.NextAlert.Before(t) {
			log.Printf("Saying \"%s\"", alert.Message)
			m.bot.Irc().Say(m.bot.Config.Channel, alert.Message)
			alert.NextAlert.Time = t.Add(alert.Period.Duration)
			m.db.Save(&alert)
		}
	}
}

func (m *AlertModule) worker() {
	for {
		ticker := time.NewTicker(1 * time.Minute)
		select {
		case t := <-ticker.C:
			m.tick(t)
		case _, ok := <-m.closeC:
			if !ok {
				return
			}
		}
	}
}

func (s *AlertService) New(r *http.Request, alert *Alert, id *int) error {
	alert.ID = 0
	return s.Update(r, alert, id)
}

func (s *AlertService) Update(r *http.Request, alert *Alert, id *int) error {
	if !s.module.bot.IsAdminRequest(r) {
		return fmt.Errorf("access denied")
	}
	err := s.module.db.Save(alert)
	if err != nil {
		*id = -1
		return err
	}

	*id = alert.ID
	return nil
}

func (s *AlertService) Get(r *http.Request, id *int, alert *Alert) error {
	return s.module.db.One("ID", *id, alert)
}

func (s *AlertService) Trigger(r *http.Request, id *int, resp *int) error {
	var alert Alert
	err := s.module.db.One("ID", *id, alert)
	if err != nil {
		return err
	}
	alert.NextAlert.Time = time.Now()
	return s.module.db.Save(alert)
}
