package metrics

import (
	"sync"
	"time"
)

type average5Metric struct {
	Last     [5]int64
	Name     string
	Category string

	cursor int
	lock   sync.Mutex
}

func (m *average5Metric) Time(started time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()
	now := time.Now()
	m.Last[m.cursor] = int64(now.Sub(started))
	m.cursor = (m.cursor + 1) % 5
}

func (m *average5Metric) Metric() string {
	return m.Name + " (rolling 5-avg)"
}
func (m *average5Metric) Compute() string {
	m.lock.Lock()
	defer m.lock.Unlock()
	var sum int64
	for _, v := range m.Last {
		sum += v
	}
	avg := sum / 5
	return time.Duration(avg).String()
}

// GetUserUIDDbTime represents the average time to query the database for a user based on UID
var GetUserUIDDbTime = &average5Metric{Name: "getUserUID", Category: "db"}

// GetSessionSIDDbTime represents the average time to query the database for a session based on SID
var GetSessionSIDDbTime = &average5Metric{Name: "getSessionSID", Category: "db"}

// GetSourcesUIDDbTime represents the average time to query the database for all a users sources.
var GetSourcesUIDDbTime = &average5Metric{Name: "getFSSourcesForUser", Category: "db"}

// GetConvosUIDDbTime represents the average time to query the database for all a users conversations.
var GetConvosUIDDbTime = &average5Metric{Name: "getConverstionsForUser", Category: "db"}

// GetMessagingSourcesUIDDbTime represents the average time to query the database for all a users messaging sources.
var GetMessagingSourcesUIDDbTime = &average5Metric{Name: "getMessagingSourcesForUser", Category: "db"}

// GetMessagesCIDDbTime represents the average time to query the database for all a conversations messages.
var GetMessagesCIDDbTime = &average5Metric{Name: "getMessagesForConversation", Category: "db"}

// InsertMessageDbTime represents the average time to insert a message into the database.
var InsertMessageDbTime = &average5Metric{Name: "insertMessage", Category: "db"}

// InsertLogDbTime represents the average time to insert a log row for an integration.
var InsertLogDbTime = &average5Metric{Name: "insertLog", Category: "db"}

// GetLogsByRunnableDbTime represents the average time to query the database for a runnables log entries.
var GetLogsByRunnableDbTime = &average5Metric{Name: "getLogsForRunnable", Category: "db"}

// GetFilteredLogsByRunnableDbTime represents the average time to filter query the database for a runnables log entries.
var GetFilteredLogsByRunnableDbTime = &average5Metric{Name: "getLogsForRunID", Category: "db"}

type metric interface {
	Compute() string
	Metric() string
}

// GetByCategory returns metrics grouped into category.
func GetByCategory() map[string][]metric {
	return map[string][]metric{
		"db": []metric{
			GetSessionSIDDbTime,
			GetUserUIDDbTime,
			GetSourcesUIDDbTime,
			GetConvosUIDDbTime,
			GetMessagingSourcesUIDDbTime,
			GetMessagesCIDDbTime,
			InsertMessageDbTime,
			InsertLogDbTime,
			GetLogsByRunnableDbTime,
			GetFilteredLogsByRunnableDbTime,
		},
	}
}
