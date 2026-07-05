module View.Components exposing (primaryButton, surface)

import Html exposing (Html, div)
import Html.Attributes exposing (class)
import Material.Button as Button


{-| Project-owned boundary around the selected Material component library.
-}
primaryButton : msg -> String -> Html msg
primaryButton onClick label =
    Button.raised
        (Button.config |> Button.setOnClick onClick)
        label


surface : List (Html msg) -> Html msg
surface children =
    div [ class "surface" ] children
