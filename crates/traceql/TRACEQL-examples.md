{} | rate()
{span.component="net/http"}|rate()
{nestedSetParent<0 && true} | rate()
{} >> {}
{ span.http.method = "GET" || span.http.method = "POST" }
{ span.http.response_code = 200 || span.http.method = "GET" }
