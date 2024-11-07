CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(128) UNIQUE NOT NULL,
    description VARCHAR(512) NOT NULL,
    location VARCHAR(255) NOT NULL,
    event_start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    event_end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX event_name_index ON event (name);

CREATE TYPE ticket_status AS ENUM ('available', 'pending', 'sold');
CREATE TABLE ticket (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES event (id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(512) NOT NULL,
    price VARCHAR(255) NOT NULL,
    benefits JSONB NOT NULL,
    status ticket_status NOT NULL DEFAULT 'available',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
CREATE INDEX ticket_name_index ON ticket (name);

CREATE TABLE ticket_inputs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES event (id) ON DELETE CASCADE,
    inputs JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TYPE attendee_status AS ENUM ('waiting', 'attended');
CREATE TABLE attendee (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID REFERENCES event (id) ON DELETE SET NULL,
    ticket_id UUID REFERENCES ticket (id) ON DELETE SET NULL,
    data JSONB NOT NULL,
    status attendee_status NOT NULL DEFAULT 'waiting',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE payment (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID REFERENCES event (id) ON DELETE SET NULL,
    data JSONB NOT NULL,
    name VARCHAR(128) NOT NULL,
    email VARCHAR(128) NOT NULL,
    bill_link_id INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
CREATE INDEX payment_name_index ON payment (name);