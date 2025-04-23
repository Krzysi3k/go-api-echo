package handlers

import (
	"context"
	"encoding/json"

	//"sort"
	//"time"

	"github.com/redis/go-redis/v9"
)

type JobOffer struct {
	Link      string `json:"link"`
	Title     string `json:"title"`
	Currency  string `json:"currency"`
	Maxsalary int    `json:"maxsalary"`
	//Published     string    `json:"published"`
	//PublishedTime time.Time `json:"-"`
}

func FetchOffers(ctx context.Context, rdb *redis.Client) []JobOffer {
	keyNames := rdb.Keys(ctx, "*job:offers*").Val()

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

	return allOffers
}
