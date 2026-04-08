# Bad Var Groups - Required Var in Required Group

This test package has a var with `required: true` inside a required var_group.
Should fail validation because vars in a var_group should not have `required: true` - the requirement is inferred from the var_group.
