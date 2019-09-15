vcl 4.1;

acl invalidators {
    "127.0.0.1";
}

sub vcl_recv {
    if (req.method == "PURGE") {
        if (!client.ip ~ invalidators) {
            return (synth(405, "Not allowed"));
        }

        return (purge);
    }
}

backend default {
  .host = "nginx";
  .port = "80";
}
