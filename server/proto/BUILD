load("@rules_proto_grpc_go//:defs.bzl", "go_grpc_library")
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

