package main

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	pb "github.com/michaelc445/proto"
)

func TestRemovePoster(t *testing.T) {

	tests := []struct {
		name       string
		userId     int32
		partyId    int32
		posterId   int32
		wantErr    bool
		returnRows *sqlmock.Rows
		wantCode   pb.ResponseCode
	}{
		{
			name:       "poster does not exist",
			userId:     1,
			partyId:    1,
			posterId:   1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}),
			wantCode:   pb.ResponseCode_FAILED,
			wantErr:    true,
		},
		{
			name:       "poster does not belong to party",
			userId:     1,
			partyId:    1,
			posterId:   1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}).AddRow(2, 1),
			wantCode:   pb.ResponseCode_FAILED,
			wantErr:    true,
		},
		{
			name:       "userId not set",
			partyId:    1,
			posterId:   1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}).AddRow(1, 1),
			wantErr:    true,
			wantCode:   pb.ResponseCode_FAILED,
		},
		{
			name:       "partyId not set",
			userId:     1,
			posterId:   1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}).AddRow(1, 1),
			wantErr:    true,
			wantCode:   pb.ResponseCode_FAILED,
		},
		{
			name:       "posterId not set",
			userId:     1,
			partyId:    1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}).AddRow(1, 1),
			wantErr:    true,
			wantCode:   pb.ResponseCode_FAILED,
		},
		{
			name:       "success",
			userId:     1,
			partyId:    1,
			posterId:   1,
			returnRows: sqlmock.NewRows([]string{"partyId", "posterId"}).AddRow(1, 1),
			wantErr:    false,
			wantCode:   pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectQuery(checkPosterQuery).WithArgs(tc.posterId).WillReturnRows(tc.returnRows)
			mock.ExpectExec("DELETE").WithArgs(tc.posterId, tc.partyId).WillReturnResult(sqlmock.NewResult(0, 0))
			res, err := server.RemovePoster(ctx, &pb.RemovePosterRequest{UserId: tc.userId, PartyId: tc.partyId, PosterId: tc.posterId})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
		})
	}
}

func TestPlacePoster(t *testing.T) {
	tests := []struct {
		name         string
		userId       int32
		partyId      int32
		location     *pb.Location
		wantErr      bool
		returnResult driver.Result
		wantCode     pb.ResponseCode
	}{
		{
			name:         "userId not set",
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "partyId not set",
			userId:       1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "Location not set",
			userId:       1,
			partyId:      1,
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      true,
			wantCode:     pb.ResponseCode_FAILED,
		},
		{
			name:         "success",
			userId:       1,
			partyId:      1,
			location:     &pb.Location{Lat: 1, Lng: 2},
			returnResult: sqlmock.NewResult(1, 2),
			wantErr:      false,
			wantCode:     pb.ResponseCode_OK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, mock, err := sqlmock.New()
			defer db.Close()
			if err != nil {
				t.Fatalf("an error occured while creating fake sql database %v", err)

			}
			server := &server{DB: db}
			mock.ExpectExec("insert").WithArgs(tc.partyId, tc.userId, tc.location.GetLat(), tc.location.GetLng()).WillReturnResult(tc.returnResult)

			res, err := server.PlacePoster(ctx, &pb.PlacementRequest{UserId: tc.userId, PartyId: tc.partyId, Location: tc.location})

			if (!tc.wantErr && err != nil) || (tc.wantErr && err == nil) {
				t.Fatalf("expected error: %v but got err: %v", tc.wantErr, err)
			}

			if res.Code != tc.wantCode {
				t.Fatalf("got code %v want code %v", res.Code, tc.wantCode)
			}
		})
	}
}
