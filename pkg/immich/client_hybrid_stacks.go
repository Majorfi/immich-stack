package immich

import (
	"sync"

	"github.com/majorfi/immich-stack/pkg/utils"
)

/**************************************************************************************************
** fetchAllStacksHybrid combines GET /assets/{id} (Phase 1, fast) with a /stacks?primaryAssetId=X
** fallback (Phase 2) for assets where Phase 1 returned no stack reference. This is the
** complete-coverage variant: Phase 1 catches all non-archived stacks fast, Phase 2 recovers
** archived primaries that /assets/{id} silently strips.
**
** Phase 2 only runs for assets reported as "no stack" by Phase 1, so the total cost is
** roughly N + (N - non-archived stacked count) HTTP calls, where N is the total asset count.
**
** @param concurrency - Number of parallel requests in flight. Defaults to 10.
**************************************************************************************************/
func (c *Client) fetchAllStacksHybrid(concurrency int) (map[string]utils.TStack, error) {
	if concurrency <= 0 {
		concurrency = 10
	}

	assetIDs, err := c.fetchAllAssetIDsViaSearch()
	if err != nil {
		return nil, err
	}
	c.logger.Infof("📋 Phase 1: %d assets via /assets/{id} (concurrency=%d)", len(assetIDs), concurrency)

	stacksByID, noStackAssetIDs, p1Failed := c.phase1Lookups(assetIDs, concurrency)
	c.logger.Infof("📚 Phase 1: %d stacks discovered, %d 'no stack' assets to verify", len(stacksByID), len(noStackAssetIDs))

	c.logger.Infof("📋 Phase 2: %d fallback lookups via /stacks?primaryAssetId", len(noStackAssetIDs))
	p2Recovered, p2Failed := c.phase2Fallback(noStackAssetIDs, concurrency, stacksByID)
	c.logger.Infof("📚 Phase 2: %d additional stacks recovered", p2Recovered)

	if p1Failed > 0 || p2Failed > 0 {
		c.logger.Warnf("⚠️  Failures — phase 1: %d, phase 2: %d", p1Failed, p2Failed)
	}

	stacksMap := make(map[string]utils.TStack)
	for _, stack := range stacksByID {
		for _, asset := range stack.Assets {
			stacksMap[asset.ID] = *stack
		}
	}

	c.logger.Infof("📚 Hybrid total: %d stacks", len(stacksByID))
	return stacksMap, nil
}

/**************************************************************************************************
** phase1Lookups fans out GET /assets/{id} for every asset ID and groups results by stack.
** Returns the partial stacks map, the list of assets that had no stack reference (candidates
** for phase 2), and the number of phase-1 failures.
**************************************************************************************************/
func (c *Client) phase1Lookups(assetIDs []string, concurrency int) (map[string]*utils.TStack, []string, int) {
	type result struct {
		asset utils.TAsset
		err   error
	}

	sem := make(chan struct{}, concurrency)
	results := make(chan result, len(assetIDs))
	var wg sync.WaitGroup

	for _, id := range assetIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(assetID string) {
			defer wg.Done()
			defer func() { <-sem }()
			asset, err := c.getAssetWithRetry(assetID, 4)
			results <- result{asset: asset, err: err}
		}(id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	stacksByID := make(map[string]*utils.TStack)
	var noStack []string
	failed := 0
	for r := range results {
		if r.err != nil {
			failed++
			continue
		}
		if r.asset.Stack == nil || r.asset.Stack.ID == "" {
			noStack = append(noStack, r.asset.ID)
			continue
		}
		existing, ok := stacksByID[r.asset.Stack.ID]
		if !ok {
			existing = &utils.TStack{
				ID:             r.asset.Stack.ID,
				PrimaryAssetID: r.asset.Stack.PrimaryAssetID,
				Assets:         []utils.TAsset{},
			}
			stacksByID[r.asset.Stack.ID] = existing
		}
		existing.Assets = append(existing.Assets, r.asset)
	}
	return stacksByID, noStack, failed
}

/**************************************************************************************************
** phase2Fallback calls /stacks?primaryAssetId=X for every asset whose phase-1 response
** lacked stack info. Newly discovered stacks (those whose primary is archived) are added
** to the shared stacksByID map. Stacks already known from phase 1 are skipped.
**************************************************************************************************/
func (c *Client) phase2Fallback(assetIDs []string, concurrency int, stacksByID map[string]*utils.TStack) (int, int) {
	if len(assetIDs) == 0 {
		return 0, 0
	}

	type result struct {
		stacks []utils.TStack
		err    error
	}

	sem := make(chan struct{}, concurrency)
	results := make(chan result, len(assetIDs))
	var wg sync.WaitGroup

	for _, id := range assetIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(assetID string) {
			defer wg.Done()
			defer func() { <-sem }()
			stacks, err := c.getStackByPrimaryWithRetry(assetID, 4)
			results <- result{stacks: stacks, err: err}
		}(id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	recovered := 0
	failed := 0
	for r := range results {
		if r.err != nil {
			failed++
			continue
		}
		for _, stack := range r.stacks {
			existing, ok := stacksByID[stack.ID]
			if !ok {
				stacksByID[stack.ID] = &utils.TStack{
					ID:             stack.ID,
					PrimaryAssetID: stack.PrimaryAssetID,
					Assets:         append([]utils.TAsset{}, stack.Assets...),
				}
				recovered++
				continue
			}
			// Stack was already partially discovered in phase 1 (e.g., via a non-archived
			// child whose /assets/{id} response carried the stack reference). Phase 2's
			// /stacks response is authoritative, so merge any members phase 1 missed —
			// typically the archived primary itself, which /assets/{id} hides.
			seen := make(map[string]bool, len(existing.Assets))
			for _, a := range existing.Assets {
				seen[a.ID] = true
			}
			for _, a := range stack.Assets {
				if !seen[a.ID] {
					existing.Assets = append(existing.Assets, a)
				}
			}
		}
	}
	return recovered, failed
}
