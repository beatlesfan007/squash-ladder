load("@gazelle//:def.bzl", "gazelle")
load("@rules_proto_grpc_js_npm//:defs.bzl", "npm_link_all_packages")

# Link all npm packages from rules_proto_grpc_js_npm
# This creates //:node_modules/* targets that js_grpc_web_library expects
npm_link_all_packages(name = "node_modules")

# Gazelle will generate BUILD files for Go packages
gazelle(
    name = "gazelle",
    prefix = "",
)
