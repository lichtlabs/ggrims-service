package events

import (
	"context"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/types/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lichtlabs/ggrims-service/events/db"
)

// ListEventAttendees List attendees on an event
//
//encore:api auth method=GET path=/v1/events/:id/attendees
func ListEventAttendees(ctx context.Context, id uuid.UUID, params *ListQuery) (*BaseResponse[[]db.ListAttendeeRow], error) {
	eb := errs.B()

	extractedParam := extractQuery(params)
	data, err := query.ListAttendee(ctx, db.ListAttendeeParams{
		OrderBy: extractedParam.OrderBy,
		Limits:  extractedParam.Limit,
		Offsets: extractedParam.Page,
		EventID: pgtype.UUID{
			Bytes: id,
			Valid: true,
		},
	})
	if err != nil {
		rlog.Error("An error occurred while retrieving attendees", "ListAttendees:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while retrieving attendees").Err()
	}

	return &BaseResponse[[]db.ListAttendeeRow]{
		Data:    data,
		Message: "Attendees retrieved successfully",
	}, nil
}
