package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

func makeFindLinksEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(FindLinksRequest)
		resp, err := s.FindLinks(ctx, &Message{URL: req.URL, Username: req.Username})
		if err != nil {
			return FindLinksResponse{Err: err}, nil
		}
		return FindLinksResponse{
			resp.Title,
			resp.Links,
			err,
		}, nil
	}
}

func makeGetListEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetListRequest)
		resp, err := s.GetList(ctx, req.Username)
		if err != nil {
			return GetListResponse{Err: err}, nil
		}
		return GetListResponse{resp, err}, nil
	}
}

type FindLinksRequest struct {
	URL      string
	Username string
}

type FindLinksResponse struct {
	Title string
	Links []Link
	Err   error
}

func (r FindLinksResponse) error() error { return r.Err }

type GetListRequest struct {
	Username string
}

type GetListResponse struct {
	Msg string
	Err error
}

func (r GetListResponse) error() error { return r.Err }
