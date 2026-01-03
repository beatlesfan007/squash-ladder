load("@rules_proto_grpc_go//:defs.bzl", "go_grpc_library")
load("@rules_go//proto:def.bzl", "go_proto_library")
load("@rules_proto//proto:defs.bzl", "proto_library")

# Expose the proto file for use in genrules
# Expose the proto file for use in genrules
filegroup(
    name = "ladder_proto_file",
    srcs = ["ladder.proto"],
    visibility = ["//visibility:public"],
)

proto_library(
    name = "ladder_proto",
    srcs = ["ladder.proto"],
    visibility = ["//visibility:public"],
)

go_grpc_library(
    name = "ladder_go_proto",
    importpath = "squash-ladder/server/gen/ladder",
    protos = [":ladder_proto"],
    visibility = ["//visibility:public"],
)

# Storage Proto - Standalone, NO dependency on ladder_proto

proto_library(
    name = "storage_proto",
    srcs = ["storage.proto"],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "storage_go_proto",
    importpath = "squash-ladder/server/gen/storage",
    proto = ":storage_proto",
    visibility = ["//visibility:public"],
)
