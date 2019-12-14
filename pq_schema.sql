CREATE TABLE files (
  id varchar(20) PRIMARY KEY,
  hash  varchar(40) DEFAULT NULL,
  originalname varchar(255) DEFAULT NULL,
  filename varchar(30) DEFAULT NULL,
  size INTEGER DEFAULT NULL,
  date DATE DEFAULT NULL,
  delid varchar(40) DEFAULT NULL,
  username INTEGER default 0,
  dir VARCHAR(2) default '00'
);
