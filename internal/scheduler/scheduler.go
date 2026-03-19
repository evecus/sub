package scheduler

import (
	"fmt"
	"time"

	"github.com/evecus/sub/internal/api"
	"github.com/evecus/sub/internal/store"
)

// Start 启动定时刷新，每天凌晨2点刷新所有远程订阅
func Start(s *store.Store) {
	go run(s)
}

func run(s *store.Store) {
	waitUntilNextRefresh()
	for {
		refresh(s)
		waitUntilNextRefresh()
	}
}

func waitUntilNextRefresh() {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if !now.Before(next) {
		next = next.Add(24 * time.Hour)
	}
	d := time.Until(next)
	fmt.Printf("[Scheduler] 下次刷新时间: %s（%.1f 小时后）\n",
		next.Format("2006-01-02 15:04:05"), d.Hours())
	time.Sleep(d)
}

func refresh(s *store.Store) {
	subs := s.GetSubscriptions()
	success := 0
	for _, sub := range subs {
		if sub.SourceType != store.SourceURL || sub.URL == "" {
			continue
		}
		nodes, err := api.FetchAndParse(sub.URL)
		if err != nil {
			fmt.Printf("[Scheduler] 刷新失败 [%s]: %v，保留旧数据\n", sub.Name, err)
			continue
		}
		api.AssignIDs(nodes)
		sub.Nodes = nodes
		sub.NodeCount = len(nodes)
		if err := s.UpdateSubscription(sub); err != nil {
			fmt.Printf("[Scheduler] 保存失败 [%s]: %v\n", sub.Name, err)
			continue
		}
		success++
	}
	total := 0
	for _, sub := range subs {
		if sub.SourceType == store.SourceURL && sub.URL != "" {
			total++
		}
	}
	fmt.Printf("[Scheduler] 凌晨2点刷新完成，成功 %d/%d 条远程订阅\n", success, total)
}
