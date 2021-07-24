#!/bin/sh

TEMP_FILE="all.txt"

touch $TEMP_FILE

echo  > $TEMP_FILE

cat server/graphql/_queries.graphql >> $TEMP_FILE
cat server/graphql/_mutations.graphql >> $TEMP_FILE
cat server/graphql/_subscriptions.graphql >> $TEMP_FILE
for filename in server/graphql/*.graphql; do
    if [[ ! "$filename" =~ _.* ]]
    then
        cat $filename >> $TEMP_FILE
        echo  >> $TEMP_FILE
    fi
done

cat server/graphql/_index.graphql >> $TEMP_FILE

sed "/# CODE GENERATED FROM SCRIPT, DO NOT CHANGE MANUALLY/r ${TEMP_FILE}" server/graphql/_template.go > server/graphql/schema.go

rm -f $TEMP_FILE