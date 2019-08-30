DROP TABLE IF EXISTS accounts;
CREATE TABLE accounts
(
    id           INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    phone_number CHAR(13)        NOT NULL UNIQUE,
    full_name    NVARCHAR(150),
    password     CHAR(64)
);

DROP TABLE IF EXISTS car;
CREATE TABLE car
(
    id      INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    user_id INT UNIQUE      NOT NULL REFERENCES accounts (id),
    model   VARCHAR(255)    NOT NULL,
    plate   VARCHAR(20)     NOT NULL UNIQUE,
    color   VARCHAR(30)     NOT NULL
);

DROP TABLE IF EXISTS wallets;
CREATE TABLE wallets
(
    id      INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    user_id INT UNIQUE      NOT NULL REFERENCES accounts (id)
);

DROP TABLE IF EXISTS spots;
CREATE TABLE spots
(
    id         INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    parking_id INT             NOT NULL REFERENCES parking (id),
    floor_id   INT             NOT NULL REFERENCES floors (id),
    number     INT             NOT NULL,
    free       BOOL            NOT NULL,
    price      REAL            NOT NULL
);

DROP TABLE IF EXISTS parking;
CREATE TABLE parking
(
    id         INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    name       VARCHAR(255)    NOT NULL UNIQUE,
    capacity   INT             NOT NULL,
    start_time TIME            NOT NULL,
    end_time   TIME            NOT NULL
);

DROP TABLE IF EXISTS floors;
CREATE TABLE floors
(
    id          INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    parking_lot INT             NOT NULL REFERENCES parking (id),
    number      INT             NOT NULL,
    capacity    INT             NOT NULL
);

DROP TABLE IF EXISTS reserves;
CREATE TABLE reserves
(
    id          INT PRIMARY KEY NOT NULL AUTO_INCREMENT,
    user_id     INT UNIQUE      NOT NULL REFERENCES accounts (id),
    car_id      INT UNIQUE      NOT NULL REFERENCES car (id),
    spot_id     INT UNIQUE      NOT NULL REFERENCES spots (id),
    start_time  TIME            NOT NULL,
    end_time    TIME            NOT NULL,
    date        DATE            NOT NULL,
    paid_online BOOL            NOT NULL,
    price       REAL            NOT NULL
);