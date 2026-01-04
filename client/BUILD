load("@aspect_rules_ts//ts:defs.bzl", "ts_project")
load("@rules_nodejs//nodejs:defs.bzl", "nodejs_binary", "npm_package")

# TypeScript project
ts_project(
    name = "client_ts",
    srcs = glob([
        "src/**/*.ts",
        "src/**/*.tsx",
    ]),
    tsconfig = "tsconfig.json",
    deps = [
        "//client/src/grpc:grpc_ts",
        "@npm//react",
        "@npm//react-dom",
    ],
)

# NPM package dependencies
npm_package(
    name = "client_package",
    package_name = "squash-ladder-client",
    srcs = [
        "package.json",
        ":client_ts",
    ] + glob(["public/**"]),
)
