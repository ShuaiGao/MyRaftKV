protoc -I=election/ election/election.proto --go_out=plugins=grpc:election/rpc