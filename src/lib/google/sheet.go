package lib_google

import (
	lib_error "application/src/lib/error"
	"context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"os"
)

type SheetClient struct {
	srv           *sheets.Service
	spreadsheetId string
}

func NewSheetClient(ctx context.Context, spreadsheetId string) (*SheetClient, error) {
	credencial := option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	srv, err := sheets.NewService(ctx, credencial)
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	return &SheetClient{
		srv:           srv,
		spreadsheetId: spreadsheetId,
	}, nil
}

func (s *SheetClient) Get(range_ string) ([][]interface{}, error) {
	resp, err := s.srv.Spreadsheets.Values.Get(s.spreadsheetId, range_).Do()
	if err != nil {
		return nil, lib_error.WrapError(err)
	}
	return resp.Values, nil
}
