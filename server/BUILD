load("@rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_library(
    name = "server_pkg",
    srcs = [
        "auth.go",
        "model.go",
        "run.go",
        "service.go",
    ],
    importpath = "squash-ladder/server",
    visibility = ["//visibility:public"],
    deps = [
        "//server/proto:ladder_go_proto",
        "//server/proto:storage_go_proto",
        "@com_github_golang_jwt_jwt_v5//:jwt",
        "@com_github_google_uuid//:uuid",
        "@com_github_improbable_eng_grpc_web//go/grpcweb",
        "@com_github_lib_pq//:pq",
        "@com_github_micahparks_keyfunc_v3//:keyfunc",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_google_grpc//codes",
        "@org_golang_google_grpc//metadata",
        "@org_golang_google_grpc//status",
        "@org_golang_google_protobuf//proto",
    ],
)

go_library(
    name = "server_lib",
    srcs = ["cmd/server/main.go"],
    importpath = "squash-ladder/server/cmd/server",
    deps = [
        ":server_pkg",
        "//server/proto:ladder_go_proto",
    ],
)

go_binary(
    name = "server",
    embed = [":server_lib"],
    visibility = ["//visibility:public"],
)

go_test(
    name = "server_test",
    srcs = [
        "model_auth_test.go",
        "model_test.go",
        "service_test.go",
    ],
    embed = [":server_pkg"],
    deps = [
        "//server/proto:ladder_go_proto",
        "@com_github_testcontainers_testcontainers_go//:go_default_library",
        "@com_github_testcontainers_testcontainers_go//wait",
        "@com_github_testcontainers_testcontainers_go_modules_postgres//:postgres",
    ],
)

go_library(
    name = "verify_lib",
    srcs = ["cmd/verify/main.go"],
    importpath = "squash-ladder/server/cmd/verify",
    deps = [
        ":server_pkg",
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
        ":server_pkg",
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
