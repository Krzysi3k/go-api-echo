package main

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

type JobOffer struct {
	Link          string    `json:"link"`
	Title         string    `json:"title"`
	Currency      string    `json:"currency"`
	Maxsalary     int       `json:"maxsalary"`
	Published     string    `json:"published"`
	PublishedTime time.Time `json:"-"`
}

// type JobOfferSlice []JobOffer

// func (a JobOfferSlice) Len() int {
// 	return len(a)
// }

// func (a JobOfferSlice) Less(i, j int) bool {
// 	return a[i].PublishedTime.Before(a[j].PublishedTime)
// }

// func (a JobOfferSlice) Swap(i, j int) {
// 	a[i], a[j] = a[j], a[i]
// }

func fetchOffers(ctx context.Context, rdb *redis.Client) []JobOffer {
	count := 50
	cursor := uint64(0)
	var keyNames []string

	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "*job:offers", int64(count)).Result()
		if err != nil {
			log.Fatal("error while scanning keys", err)
		}
		keyNames = append(keyNames, keys...)

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	allOffers := make([]JobOffer, 0)
	result := rdb.MGet(ctx, keyNames...).Val()
	for _, v := range result {
		if payload, ok := v.(string); ok {
			var offers []JobOffer
			// var offers JobOfferSlice
			json.Unmarshal([]byte(payload), &offers)
			allOffers = append(allOffers, offers...)
		}
	}

	for idx, o := range allOffers {
		date, err := time.Parse("2006-01-02", o.Published)
		if err != nil {
			log.Fatal("Error parsing date:", err)
		}
		allOffers[idx].PublishedTime = date
	}

	sort.Slice(allOffers, func(i, j int) bool {
		return allOffers[i].PublishedTime.After(allOffers[j].PublishedTime)
	})

	return allOffers
}
