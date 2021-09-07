docker-compose down

docker-compose up -d 

# postgres takes a sec to spin up and accept connections
sleep 5

docker exec newsdb \
  bash -c "cat /usr/src/dump.sql | psql \
    -U root \
    -d localdb" 
