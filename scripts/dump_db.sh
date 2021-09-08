# Populate w/ current read replica
export RDSHOST="newsfeed-db-dev.c3bzqjvxdcd7.us-west-1.rds.amazonaws.com"

# Expects you to have prod db pw :|, command will prompt
# use --exclude-table-data TABLENAME to have table schema only
docker run -i --rm postgres pg_dump \
  -h $RDSHOST \
  -p 5432 \
  -U root \
  -W -d dev_jamie > dump.sql
