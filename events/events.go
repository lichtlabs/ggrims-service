package events

import (
	"context"
	"encoding/json"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
	"encore.dev/types/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lichtlabs/ggrims-service/events/db"
)

var (
	ggrimsDB = sqldb.NewDatabase("ggrims", sqldb.DatabaseConfig{
		Migrations: "./db/migrations",
	})

	pgxDB = sqldb.Driver(ggrimsDB)
	query = db.New(pgxDB)
)

var secrets struct {
	FlipApiBaseEndpoint string `json:"flip_api_base_endpoint"`
	FlipValidationToken string `json:"flip_validation_token"`
	FlipApiSecretKey    string `json:"flip_api_secret_key"`
}

type CreateEventRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Location       string              `json:"location"`
	EventStartDate time.Time           `json:"event_start_date"`
	EventEndDate   time.Time           `json:"event_end_date"`
	Inputs         []*EventTicketInput `json:"inputs"`
}

// CreateEvent Create an event
//
//encore:api auth method=POST path=/v1/events
func CreateEvent(ctx context.Context, req *CreateEventRequest) (*BaseResponse[InsertionResponse], error) {
	eb := errs.B()

	eventId, err := query.InsertEvent(ctx, db.InsertEventParams{
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		EventStartDate: pgtype.Timestamptz{
			Time:  req.EventStartDate,
			Valid: true,
		},
		EventEndDate: pgtype.Timestamptz{
			Time:  req.EventEndDate,
			Valid: true,
		},
	})
	if err != nil {
		rlog.Error("An error occurred while creating event", "CreateEvent:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while creating event").Err()
	}

	bInputs, err := json.Marshal(req.Inputs)
	if err != nil {
		rlog.Error("An error occurred while creating event", "CreateEvent:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while creating event").Err()
	}

	_, err = query.InsertEventTicketInput(ctx, db.InsertEventTicketInputParams{
		EventID: eventId,
		Inputs:  bInputs,
	})
	if err != nil {
		rlog.Error("An error occurred while creating event", "CreateEvent:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while creating event").Err()
	}

	return &BaseResponse[InsertionResponse]{
		Data: InsertionResponse{
			Created: 1,
		},
		Message: "Event created successfully",
	}, nil
}

// UpdateEvent Update an event
//
//encore:api auth method=PUT path=/v1/events/:id
func UpdateEvent(ctx context.Context, id uuid.UUID, req *db.UpdateEventParams) error {
	eb := errs.B()

	err := query.UpdateEvent(ctx, db.UpdateEventParams{
		Name:           req.Name,
		Description:    req.Description,
		Location:       req.Location,
		EventStartDate: req.EventStartDate,
		EventEndDate:   req.EventEndDate,
		EventID: pgtype.UUID{
			Bytes: id,
			Valid: true,
		},
	})
	if err != nil {
		rlog.Error("An error occurred while updating event", "UpdateEvent:err", err.Error())
		return eb.Code(errs.Internal).Msg("An error occurred while updating event").Err()
	}

	return nil
}

// DeleteEvent Delete an event
//
//encore:api auth method=DELETE path=/v1/events/:id
func DeleteEvent(ctx context.Context, id uuid.UUID) error {
	eb := errs.B()

	err := query.DeleteEvent(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	})
	if err != nil {
		rlog.Error("An error occurred while deleting event", "DeleteEvent:err", err.Error())
		return eb.Code(errs.Internal).Msg("An error occurred while deleting event").Err()
	}

	return nil
}

// GetEvent Get an event including ticket inputs
//
//encore:api public method=GET path=/v1/events/:id
func GetEvent(ctx context.Context, id uuid.UUID) (*BaseResponse[Event], error) {
	eb := errs.B()

	data, err := query.GetEvent(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	})
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while retrieving event").Err()
	}

	ticketInputs := make([]*EventTicketInput, 0)
	err = json.Unmarshal(data.TicketInputs, &ticketInputs)
	if err != nil {
		rlog.Error("An error occurred while decoding ticket inputs", "GetEvent:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while decoding ticket inputs").Err()
	}

	return &BaseResponse[Event]{
		Data: Event{
			ID:             data.ID,
			Name:           data.Name,
			Description:    data.Description,
			Location:       data.Location,
			EventStartDate: data.EventStartDate,
			EventEndDate:   data.EventEndDate,
			CreatedAt:      data.CreatedAt,
			UpdatedAt:      data.UpdatedAt,
			TicketInputs:   ticketInputs,
		},
		Message: "Event retrieved successfully",
	}, nil
}

// ListEvents Get all events including ticket inputs
//
//encore:api public method=GET path=/v1/events
func ListEvents(ctx context.Context, params *ListQuery) (*BaseResponse[[]Event], error) {
	eb := errs.B()

	extractedParam := extractQuery(params)
	rlog.Info("ListEvents", "extractedParam", extractedParam)
	data, err := query.ListEvent(ctx, db.ListEventParams{
		OrderBy: extractedParam.OrderBy,
		Offsets: extractedParam.Page,
		Limits:  extractedParam.Limit,
	})
	if err != nil {
		rlog.Error("An error occurred while retrieving events", "ListEvents:err", err.Error())
		return nil, eb.Code(errs.Internal).Msg("An error occurred while retrieving events").Err()
	}

	events := make([]Event, 0)
	for _, data := range data {
		ticketInputs := make([]*EventTicketInput, 0)
		err := json.Unmarshal(data.TicketInputs, &ticketInputs)
		if err != nil {
			rlog.Error("An error occurred while decoding ticket inputs", "ListEvents:err", err.Error())
			return nil, eb.Code(errs.Internal).Msg("An error occurred while decoding ticket inputs").Err()
		}

		events = append(events, Event{
			ID:             data.ID,
			Name:           data.Name,
			Description:    data.Description,
			Location:       data.Location,
			EventStartDate: data.EventStartDate,
			EventEndDate:   data.EventEndDate,
			CreatedAt:      data.CreatedAt,
			UpdatedAt:      data.UpdatedAt,
			TicketInputs:   ticketInputs,
		})
	}

	return &BaseResponse[[]Event]{
		Data:    events,
		Message: "Events retrieved successfully",
	}, nil
}

// ListUpcomingEvents Get all upcoming events including ticket inputs
//
//encore:api public method=GET path=/v1/upcoming-events
func ListUpcomingEvents(ctx context.Context) (*BaseResponse[[]Event], error) {
	eb := errs.B()

	data, err := query.ListUpcomingEvent(ctx)
	if err != nil {
		return nil, eb.Code(errs.Internal).Msg(err.Error()).Err()
	}

	events := make([]Event, 0)
	for _, data := range data {
		events = append(events, Event{
			ID:             data.ID,
			Name:           data.Name,
			Description:    data.Description,
			Location:       data.Location,
			EventStartDate: data.EventStartDate,
			EventEndDate:   data.EventEndDate,
			CreatedAt:      data.CreatedAt,
			UpdatedAt:      data.UpdatedAt,
		})
	}

	return &BaseResponse[[]Event]{
		Data:    events,
		Message: "Events retrieved successfully",
	}, nil
}
