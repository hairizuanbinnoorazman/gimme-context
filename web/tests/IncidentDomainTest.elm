module IncidentDomainTest exposing (tests)

import Expect
import IncidentDomain exposing (nextLifecycle)
import Test exposing (Test, describe, test)

tests : Test
tests =
    describe "manual incident lifecycle"
        [ test "advances through every normal state" <|
            \_ ->
                [ "open", "investigating", "mitigating", "monitoring", "resolved", "reviewed" ]
                    |> List.map nextLifecycle
                    |> Expect.equal [ "investigating", "mitigating", "monitoring", "resolved", "reviewed", "archived" ]
        , test "does not advance branch states implicitly" <|
            \_ ->
                [ "archived", "cancelled", "dormant" ]
                    |> List.map nextLifecycle
                    |> Expect.equal [ "archived", "cancelled", "dormant" ]
        ]
