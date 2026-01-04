load("@aspect_rules_ts//ts:defs.bzl", "ts_project")
load("@rules_proto_grpc_js//:defs.bzl", "js_grpc_web_library", "js_proto_library")

# Generate JavaScript/TypeScript code from proto files using rules_proto_grpc
# js_proto_library generates both JavaScript and TypeScript definitions
js_proto_library(
    name = "ladder_proto_ts",
    protos = ["//server/proto:ladder_proto"],
)

# Generate gRPC-Web service client code from proto files
# This generates the service stubs for making gRPC-Web calls
js_grpc_web_library(
    name = "ladder_grpc_web",
    protos = ["//server/proto:ladder_proto"],
    deps = [
        ":ladder_proto_ts",
    ],
)

# TypeScript project that uses the generated proto files
ts_project(
    name = "grpc_ts",
    srcs = [
        "ladderService.ts",
    ],
    tsconfig = "//client:tsconfig.json",
    visibility = ["//visibility:public"],
    deps = [
        ":ladder_grpc_web",
        ":ladder_proto_ts",
        "@npm//@improbable-eng/grpc-web",
        "@npm//google-protobuf",
    ],
)
