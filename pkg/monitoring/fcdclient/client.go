package fcdclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/ratelimit"
)

func New(
	fcdURL string,
	requestsPerSec int,
) Client {
	return &client{
		fcdURL,
		&http.Client{},
		ratelimit.New(int(requestsPerSec),
			ratelimit.Per(1*time.Second), // one request every 1 second
			ratelimit.WithoutSlack,       // don't accumulate previously "unspent" requests for future bursts
		),
	}
}

type client struct {
	fcdURL     string
	httpClient *http.Client
	limiter    ratelimit.Limiter
}

func (c *client) GetTxList(ctx context.Context, params GetTxListParams) (Response, error) {
	_ = c.limiter.Take()
	query := url.Values{}
	if params.Account.String() != "" {
		query.Set("account", params.Account.String())
	}
	if params.Limit != 0 {
		query.Set("limit", strconv.Itoa(params.Offset))
	}
	if params.Offset != 0 {
		query.Set("offset", strconv.Itoa(params.Limit))
	}
	if params.Block != "" {
		query.Set("block", params.Block)
	}
	getTxsURL, err := url.Parse(c.fcdURL)
	if err != nil {
		return Response{}, err
	}
	getTxsURL.Path = "/v1/txs"
	getTxsURL.RawQuery = query.Encode()
	readTxsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, getTxsURL.String(), nil)
	if err != nil {
		return Response{}, fmt.Errorf("unable to build a request to the terra FCD: %w", err)
	}
	res, err := c.httpClient.Do(readTxsReq)
	if err != nil {
		return Response{}, fmt.Errorf("unable to fetch transactions from terra FCD: %w", err)
	}
	defer res.Body.Close()
	// Decode the response
	output := Response{}
	resBody, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(resBody, &output); err != nil {
		return Response{}, fmt.Errorf("unable to decode transactions from response '%s': %w", resBody, err)
	}
	return output, nil
}
