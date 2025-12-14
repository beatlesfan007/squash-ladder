load("@rules_proto_grpc_go//:defs.bzl", "go_grpc_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

# Expose the proto file for use in genrules
filegroup(
    name = "players_proto_file",
    srcs = ["players.proto"],
    visibility = ["//visibility:public"],
)

proto_library(
    name = "players_proto",
    srcs = ["players.proto"],
    visibility = ["//visibility:public"],
)

go_grpc_library(
    name = "players_go_proto",
    importpath = "squash-ladder/server/gen/players",
    protos = [":players_proto"],
    visibility = ["//visibility:public"],
)

