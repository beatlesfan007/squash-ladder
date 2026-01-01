load("@rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "handlers_lib",
    srcs = [
        "handlers/ladder.go",
        "handlers/players.go",
    ],
    importpath = "squash-ladder/server/handlers",
    deps = [
        "//server/proto:players_go_proto",
        "@com_github_google_uuid//:uuid",
    ],
    visibility = ["//visibility:public"],
)

go_library(
    name = "server_lib",
    srcs = ["cmd/server/main.go"],
    importpath = "squash-ladder/server/cmd/server",
    deps = [
        "//server/proto:players_go_proto",
        ":handlers_lib",
        "@org_golang_google_grpc//:go_default_library",
        "@com_github_improbable_eng_grpc_web//go/grpcweb",
    ],
)

go_binary(
    name = "server",
    embed = [":server_lib"],
    visibility = ["//visibility:public"],
)

