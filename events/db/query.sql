-- ###############################################################
-- Event
-- ###############################################################

-- name: InsertEvent :one
INSERT INTO event
    (name, description, location, event_start_date, event_end_date)
VALUES
    (@name, @description, @location, @event_start_date, @event_end_date)
RETURNING id;

-- name: UpdateEvent :exec
UPDATE event
SET
    name = @name,
    description = @description,
    location = @location,
    event_start_date = @event_start_date,
    event_end_date = @event_end_date
WHERE id = @event_id;

-- name: DeleteEvent :exec
DELETE FROM event
WHERE id = $1;

-- name: GetEvent :one
SELECT
    e.id,
    e.name,
    e.description,
    e.location,
    e.event_start_date,
    e.event_end_date,
    e.created_at,
    e.updated_at,
    eti.inputs as ticket_inputs
FROM event e
LEFT JOIN ticket_inputs eti ON e.id = eti.event_id
WHERE e.id = $1;

-- name: ListEvent :many
SELECT
    event.id,
    event.name,
    event.description,
    event.location,
    event.event_start_date,
    event.event_end_date,
    event.created_at,
    event.updated_at,
    ticket_inputs.inputs as ticket_inputs
FROM event
LEFT JOIN ticket_inputs ticket_inputs ON event.id = ticket_inputs.event_id
ORDER BY @order_by
OFFSET @offsets
LIMIT @limits;

-- name: ListUpcomingEvent :many
SELECT id, name, description, location, event_start_date, event_end_date, created_at, updated_at
FROM event
WHERE NOW() < event_start_date::date
ORDER BY event_start_date ASC;

-- ###############################################################
-- TicketInputs
-- ###############################################################

-- name: InsertEventTicketInput :one
INSERT INTO ticket_inputs
    (event_id, inputs)
VALUES
    (@event_id, @inputs)
RETURNING id;

-- ###############################################################
-- Ticket
-- ###############################################################

-- name: InsertTicket :one
INSERT INTO ticket
    (event_id, name, description, price, benefits, hash, min, max)
VALUES
    (@event_id, @name, @description, @price, @benefits, @hash, @min, @max)
RETURNING id;

-- name: UpdateTicket :exec
UPDATE ticket
SET
    name = @name,
    description = @description,
    price = @price,
    benefits = @benefits
WHERE event_id = @event_id;

-- name: DeleteTicket :exec
WITH rows_to_delete AS (
    SELECT *
    FROM ticket
    WHERE ticket.event_id = @ev_id AND ticket.name = @ticket_name
    LIMIT @limits
)
DELETE FROM ticket
WHERE id IN (SELECT id FROM rows_to_delete);

-- name: GetTicket :one
SELECT
    *
FROM ticket
WHERE id = $1;

-- name: ListDistinctTicket :many
SELECT DISTINCT ON (name)
    event_id,
    name,
    description,
    price,
    benefits,
    status,
    min,
    max,
    created_at,
    updated_at,
    COUNT(*) OVER (PARTITION BY name)
FROM ticket
WHERE event_id = $1 AND status = 'available'
ORDER BY name, created_at DESC;

-- name: GetAvailableTickets :many
SELECT
    id,
    name,
    price,
    hash,
    COUNT(name) AS count
FROM ticket
WHERE status = 'available' AND name = @name
GROUP BY id
LIMIT @limits;

-- name: ChangeTicketsStatus :exec
UPDATE ticket
    SET status = @status
WHERE id = @ticket_id;

-- ###############################################################
-- Attendee
-- ###############################################################

-- name: InsertAttendee :one
INSERT INTO attendee
    (event_id, ticket_id, data)
VALUES
    (@event_id, @ticket_id, @data)
RETURNING id;

-- name: UpdateAttendeeStatus :exec
UPDATE attendee
SET
    status = @status
WHERE id = @attendee_id;

-- name: DeleteAttendee :exec
DELETE FROM attendee
WHERE id = $1;

-- name: ListAttendee :many
SELECT
    e.id,
    e.event_id,
    e.ticket_id,
    e.data,
    e.created_at,
    e.updated_at
FROM attendee e
WHERE e.event_id = @event_id
ORDER BY @order_by
OFFSET @offsets
LIMIT @limits;

-- ###############################################################
-- Payment
-- ###############################################################

-- name: InsertPayment :one
INSERT INTO payment
    (event_id, data, name, email, bill_link_id)
VALUES
    (@event_id, @data, @name, @email, @bill_link_id)
RETURNING id;

-- name: UpdatePayment :exec
UPDATE payment
SET
    data = @data,
    name = @name,
    email = @email,
    bill_link_id = @bill_link_id
WHERE id = @payment_id;

-- name: DeletePayment :exec
DELETE FROM payment
WHERE id = $1;

-- name: GetPayment :one
SELECT
    e.id,
    e.event_id,
    e.data,
    e.name,
    e.email,
    e.bill_link_id,
    e.created_at,
    e.updated_at
FROM payment e
WHERE e.id = $1;

-- name: CheckPaymentExists :one
SELECT EXISTS(SELECT 1 FROM payment WHERE bill_link_id = $1) AS payment_exists;

-- name: ListPayment :many
SELECT
    e.id,
    e.event_id,
    e.data,
    e.name,
    e.email,
    e.bill_link_id,
    e.created_at,
    e.updated_at
FROM payment e
ORDER BY @order_by
OFFSET @offsets
LIMIT @limits;
