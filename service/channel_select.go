package service

import (
	"errors"
	"math/rand"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx                  *gin.Context
	TokenGroup           string
	ModelName            string
	RequestPath          string
	Retry                *int
	EnumerateCandidates  bool
	resetNextTry         bool
	candidates           []retryCandidate
	candidateIndex       int
	candidateInitialized bool
	SelectedKeyIndex     int
}

type retryCandidate struct {
	channel  *model.Channel
	group    string
	keyIndex int
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

func (p *RetryParam) HasRemainingCandidates() bool {
	return p.candidateInitialized && p.candidateIndex < len(p.candidates)
}

// AdvanceRetry moves to the next channel/key candidate. Retry is incremented
// only after the complete candidate list has failed once.
func (p *RetryParam) AdvanceRetry(maxRetries int) bool {
	if !p.candidateInitialized {
		if p.Retry == nil {
			p.Retry = new(int)
		}
		*p.Retry++
		return p.GetRetry() <= maxRetries
	}
	// RetryTimes=0 means the initial candidate is the only attempt. For a
	// request already in its final retry round, remaining candidates in that
	// round are still eligible; the next round is not.
	if maxRetries <= 0 {
		return false
	}
	if p.GetRetry() >= maxRetries {
		return p.HasRemainingCandidates()
	}
	if p.candidateIndex < len(p.candidates) {
		return true
	}
	p.candidates = nil
	p.candidateIndex = 0
	p.candidateInitialized = false
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
	return p.GetRetry() <= maxRetries
}

// CacheGetRandomSatisfiedChannel enumerates channel/key candidates. A retry
// counter represents a complete pass over the candidate list, not one key.
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	if !param.EnumerateCandidates {
		return cacheGetInitialRandomChannel(param)
	}
	if param.candidateInitialized {
		for param.candidateIndex < len(param.candidates) {
			candidate := param.candidates[param.candidateIndex]
			param.candidateIndex++
			if candidate.channel.Status != common.ChannelStatusEnabled || len(candidate.channel.GetEnabledKeyIndexes()) == 0 {
				continue
			}
			param.SelectedKeyIndex = candidate.keyIndex
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, candidate.group)
			return candidate.channel, candidate.group, nil
		}
		return nil, param.TokenGroup, errors.New("all channel/key candidates exhausted")
	}

	groups := []string{param.TokenGroup}
	if param.TokenGroup == "auto" {
		userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
		groups = GetRequestAutoGroups(param.Ctx, userGroup)
		if len(groups) == 0 {
			return nil, param.TokenGroup, errors.New("auto groups is not enabled")
		}
		if !common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry) {
			groups = groups[:1]
		}
	}
	for _, group := range groups {
		channels, err := model.GetSatisfiedChannels(group, param.ModelName, param.RequestPath)
		if err != nil {
			return nil, group, err
		}
		sort.SliceStable(channels, func(i, j int) bool {
			return channels[i].GetPriority() > channels[j].GetPriority()
		})
		for start := 0; start < len(channels); {
			end := start + 1
			for end < len(channels) && channels[end].GetPriority() == channels[start].GetPriority() {
				end++
			}
			// Keep priority ordering while randomizing the first channel within
			// a priority, preserving normal load balancing across requests.
			rand.Shuffle(end-start, func(i, j int) {
				channels[start+i], channels[start+j] = channels[start+j], channels[start+i]
			})
			start = end
		}
		for _, channel := range channels {
			for _, keyIndex := range channel.GetEnabledKeyIndexes() {
				param.candidates = append(param.candidates, retryCandidate{channel: channel, group: group, keyIndex: keyIndex})
			}
		}
	}
	param.candidateInitialized = true
	return CacheGetRandomSatisfiedChannel(param)
}

// cacheGetInitialRandomChannel preserves normal load-balancing for the first
// middleware selection. Relay retry loops opt into candidate enumeration.
func cacheGetInitialRandomChannel(param *RetryParam) (*model.Channel, string, error) {
	groups := []string{param.TokenGroup}
	if param.TokenGroup == "auto" {
		userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
		groups = GetRequestAutoGroups(param.Ctx, userGroup)
		if len(groups) == 0 {
			return nil, param.TokenGroup, errors.New("auto groups is not enabled")
		}
		start := 0
		if value, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if index, ok := value.(int); ok && index >= 0 && index < len(groups) {
				start = index
			}
		}
		for i := start; i < len(groups); i++ {
			retry := param.GetRetry()
			if i > start {
				retry = 0
			}
			channel, err := model.GetRandomSatisfiedChannel(groups[i], param.ModelName, retry, param.RequestPath)
			if err != nil {
				return nil, groups[i], err
			}
			if channel == nil {
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, groups[i])
			if common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry) && param.GetRetry() >= common.RetryTimes {
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
			}
			return channel, groups[i], nil
		}
		return nil, param.TokenGroup, nil
	}
	channel, err := model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, param.GetRetry(), param.RequestPath)
	return channel, param.TokenGroup, err
}
