# Bad Var Groups - Required Var in Optional Group

This test package has a var with `required: true` inside an optional var_group (required: false).
Should fail validation because vars in a var_group should not have `required: true` - when the var_group is optional, the entire group is optional.
