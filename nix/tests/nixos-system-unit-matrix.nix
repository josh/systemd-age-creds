{ lib }:
lib.attrsets.cartesianProduct {
  creds = [
    {
      name = "one";
      value = {
        foo = "42";
      };
    }
    {
      name = "few";
      value = builtins.listToAttrs (
        builtins.genList (i: {
          name = "foo-${builtins.toString i}";
          value = builtins.toString i;
        }) 3
      );
    }
    {
      name = "many";
      value = builtins.listToAttrs (
        builtins.genList (i: {
          name = "foo-${builtins.toString i}";
          value = builtins.toString i;
        }) 50
      );
    }
  ];
  accept = [
    {
      name = "no";
      value = false;
    }
    {
      name = "yes";
      value = true;
    }
  ];
}
