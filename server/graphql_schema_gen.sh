#!/bin/sh

TEMP_FILE="all.txt"

touch $TEMP_FILE

echo  > $TEMP_FILE

cat graphql/_queries.graphql >> $TEMP_FILE
cat graphql/_mutations.graphql >> $TEMP_FILE
cat graphql/_subscriptions.graphql >> $TEMP_FILE
for filename in graphql/*.graphql; do
    if [[ ! "$filename" =~ _.* ]]
    then
        cat $filename >> $TEMP_FILE
        echo  >> $TEMP_FILE
    fi
done

cat graphql/_index.graphql >> $TEMP_FILE

sed "/# CODE GENERATED FROM SCRIPT, DO NOT CHANGE MANUALLY/r ${TEMP_FILE}" graphql/_template.go > graphql/schema.go

rm -f $TEMP_FILE