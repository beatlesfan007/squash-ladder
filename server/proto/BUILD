load("@rules_proto//proto:defs.bzl", "proto_library")
load("@rules_go//go:def.bzl", "go_library")
load("@rules_go//proto:def.bzl", "go_proto_library")

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

go_proto_library(
    name = "players_go_proto",
    importpath = "squash-ladder/server/gen/players",
    proto = ":players_proto",
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "players_grpc_go_proto",
    importpath = "squash-ladder/server/gen/players",
    proto = ":players_proto",
    compilers = ["@rules_go//proto:go_grpc"],
    visibility = ["//visibility:public"],
)

