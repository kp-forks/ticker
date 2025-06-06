package poller

import (
	"context"
	"errors"
	"time"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/monitor/coinbase/unary"
)

type Poller struct {
	refreshInterval      time.Duration
	symbols              []string
	isStarted            bool
	ctx                  context.Context
	cancel               context.CancelFunc
	unaryAPI             *unary.UnaryAPI
	chanUpdateAssetQuote chan c.MessageUpdate[c.AssetQuote]
	chanError            chan error
	versionVector        int
}

type PollerConfig struct {
	UnaryAPI             *unary.UnaryAPI
	ChanUpdateAssetQuote chan c.MessageUpdate[c.AssetQuote]
	ChanError            chan error
}

func NewPoller(ctx context.Context, config PollerConfig) *Poller {
	ctx, cancel := context.WithCancel(ctx)

	return &Poller{
		refreshInterval:      0,
		isStarted:            false,
		ctx:                  ctx,
		cancel:               cancel,
		unaryAPI:             config.UnaryAPI,
		chanUpdateAssetQuote: config.ChanUpdateAssetQuote,
		chanError:            config.ChanError,
		versionVector:        0,
	}
}

func (p *Poller) SetSymbols(symbols []string, versionVector int) {
	p.symbols = symbols
	p.versionVector = versionVector
}

func (p *Poller) SetRefreshInterval(interval time.Duration) error {

	if p.isStarted {
		return errors.New("cannot set refresh interval while poller is started")
	}

	p.refreshInterval = interval

	return nil
}

func (p *Poller) Start() error {
	if p.isStarted {
		return errors.New("poller already started")
	}

	if p.refreshInterval <= 0 {
		return errors.New("refresh interval is not set")
	}

	p.isStarted = true

	// Start polling goroutine
	go func() {
		ticker := time.NewTicker(p.refreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				if len(p.symbols) == 0 {

					continue
				}
				versionVector := p.versionVector
				assetQuotes, _, err := p.unaryAPI.GetAssetQuotes(p.symbols)
				if err != nil {
					p.chanError <- err

					continue
				}

				for _, assetQuote := range assetQuotes {
					p.chanUpdateAssetQuote <- c.MessageUpdate[c.AssetQuote]{
						ID:            assetQuote.Meta.SymbolInSourceAPI,
						Data:          assetQuote,
						VersionVector: versionVector,
					}
				}
			}
		}
	}()

	return nil
}
