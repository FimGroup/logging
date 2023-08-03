package logging_test

import (
	"testing"

	"github.com/FimGroup/logging"

	"github.com/sirupsen/logrus"
)

func TestSampleHook(t *testing.T) {
	manager, err := logging.NewLoggerManager("logs/fim", 1, 1*1024*1024, 2, logrus.TraceLevel, true, true)
	if err != nil {
		t.Fatal(err)
	}
	logger := manager.GetLogger("demoLogger001")
	logger.WarnF("this is a demo warning logging message. parameter 01=[%s]", t.Name())

	manager2 := logging.NewNoFileLoggerManager(logrus.InfoLevel)
	logger2 := manager2.GetLogger("demoLogger002")
	logger2.WarnF("this is a demo warning logging message. parameter 01=[%s]", t.Name())
}

func TestKeepLogging(t *testing.T) {
	manager, err := logging.NewLoggerManager("logs/fim", 1, 1*1024*1024, 10, logrus.TraceLevel, true, false)
	if err != nil {
		t.Fatal(err)
	}
	logger := manager.GetLogger("demoLogger001")

	//const cnt = 10000000
	const cnt = 100
	for i := 0; i < cnt; i++ {
		logger.WarnF("this is a demo warning logging message. parameter 01=[%s]", t.Name())
		//time.Sleep(500 * time.Millisecond)
	}
}
