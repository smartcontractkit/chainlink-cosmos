// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	query "github.com/cosmos/cosmos-sdk/types/query"

	tx "github.com/cosmos/cosmos-sdk/types/tx"

	types "github.com/cosmos/cosmos-sdk/types"
)

// ChainReader is an autogenerated mock type for the ChainReader type
type ChainReader struct {
	mock.Mock
}

// ContractState provides a mock function with given fields: ctx, contractAddress, queryMsg
func (_m *ChainReader) ContractState(ctx context.Context, contractAddress types.AccAddress, queryMsg []byte) ([]byte, error) {
	ret := _m.Called(ctx, contractAddress, queryMsg)

	if len(ret) == 0 {
		panic("no return value specified for ContractState")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, []byte) ([]byte, error)); ok {
		return rf(ctx, contractAddress, queryMsg)
	}
	if rf, ok := ret.Get(0).(func(context.Context, types.AccAddress, []byte) []byte); ok {
		r0 = rf(ctx, contractAddress, queryMsg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, types.AccAddress, []byte) error); ok {
		r1 = rf(ctx, contractAddress, queryMsg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TxsEvents provides a mock function with given fields: ctx, events, paginationParams
func (_m *ChainReader) TxsEvents(ctx context.Context, events []string, paginationParams *query.PageRequest) (*tx.GetTxsEventResponse, error) {
	ret := _m.Called(ctx, events, paginationParams)

	if len(ret) == 0 {
		panic("no return value specified for TxsEvents")
	}

	var r0 *tx.GetTxsEventResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []string, *query.PageRequest) (*tx.GetTxsEventResponse, error)); ok {
		return rf(ctx, events, paginationParams)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []string, *query.PageRequest) *tx.GetTxsEventResponse); ok {
		r0 = rf(ctx, events, paginationParams)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tx.GetTxsEventResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []string, *query.PageRequest) error); ok {
		r1 = rf(ctx, events, paginationParams)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewChainReader creates a new instance of ChainReader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewChainReader(t interface {
	mock.TestingT
	Cleanup(func())
}) *ChainReader {
	mock := &ChainReader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
