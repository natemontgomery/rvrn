CREATE TABLE songs(
  Sid INT PRIMARY KEY NOT NULL,
  artist_id VARCHAR(255),
  artist_name VARCHAR(255),
  title VARCHAR(255),
  id VARCHAR(255),
  Album VARCHAR(255),
  Comment VARCHAR(255),
  Genre VARCHAR(255),
  Year INT,
  track_number INT,
  Length INT,
  Bitrate INT,
  Samplerate INT,
  Channels INT,
  Created TIMESTAMP
);