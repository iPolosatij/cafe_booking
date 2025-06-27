развернуть БД
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo service postgresql start
sudo service postgresql status

запустить psql
which psql
sudo -u postgres psql или psql postgres
(enter  pass)

создать БД
postgres=# CREATE DATABASE cafe_booking;
postgres=# CREATE USER cafe_user WITH PASSWORD '90OPKL!@qw';
postgres=# GRANT ALL PRIVILEGES ON DATABASE cafe_booking TO cafe_user;
\q

в корне проекта go run .