module IncidentDomain exposing (nextLifecycle)

nextLifecycle : String -> String
nextLifecycle current =
    case current of
        "open" -> "investigating"
        "investigating" -> "mitigating"
        "mitigating" -> "monitoring"
        "monitoring" -> "resolved"
        "resolved" -> "reviewed"
        "reviewed" -> "archived"
        _ -> current
