load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "server_pkg",
    srcs = [
        "model.go",
        "service.go",
    ],
    importpath = "squash-ladder/server",
    visibility = ["//visibility:public"],
    deps = [
        "//server/proto:ladder_go_proto",
        "@com_github_google_uuid//:uuid",
    ],
)

go_library(
    name = "server_lib",
    srcs = ["cmd/server/main.go"],
    importpath = "squash-ladder/server/cmd/server",
    deps = [
        ":server_pkg",
        "//server/proto:ladder_go_proto",
        "@com_github_improbable_eng_grpc_web//go/grpcweb",
        "@org_golang_google_grpc//:go_default_library",
    ],
)

go_binary(
    name = "server",
    embed = [":server_lib"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "verify_lib",
    srcs = ["cmd/verify/main.go"],
    importpath = "squash-ladder/server/cmd/verify",
    deps = [
        "//server/proto:ladder_go_proto",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//credentials/insecure",
    ],
    visibility = ["//visibility:private"],
)

go_binary(
    name = "verify",
    embed = [":verify_lib"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "verify_recent_lib",
    srcs = ["cmd/verify_recent/main.go"],
    importpath = "squash-ladder/server/cmd/verify_recent",
    deps = [
        "//server/proto:ladder_go_proto",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//credentials/insecure",
    ],
    visibility = ["//visibility:private"],
)

go_binary(
    name = "verify_recent",
    embed = [":verify_recent_lib"],
    visibility = ["//visibility:public"],
)
