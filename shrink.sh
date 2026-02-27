sqlite3 $1 "update requests set request_body='{}'; update requests set response_body=NULL; update request_executions set request_body='{}'; update request_executions set response_body=NULL; VACUUM;"

#sqlite3 $1 "update requests set request_body='{}';"
#sqlite3 $1 "update requests set response_body=NULL;"
#sqlite3 $1 "update request_executions set request_body='{}';"
#sqlite3 $1 "update request_executions set response_body=NULL;"
#sqlite3 $1 "VACUUM;"
