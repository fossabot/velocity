module Component.BuildOutput exposing (Model, Msg, init, view, update, events, leaveChannels, subscriptions)

{- A stateful BuildOutput component.
   I plan to convert this to a stateless component soon.
-}
-- INTERNAL

import Context exposing (Context)
import Data.Build as Build exposing (Build)
import Data.BuildStep as BuildStep exposing (BuildStep)
import Data.BuildStream as BuildStream exposing (Id, BuildStream, BuildStreamOutput)
import Data.AuthToken as AuthToken exposing (AuthToken)
import Data.Task as ProjectTask exposing (Step(..), Parameter(..))
import Request.Build
import Request.Errors
import Util exposing ((=>))
import Page.Helpers exposing (formatDateTime, formatTimeSeconds)
import Views.Build exposing (..)
import Ports


-- EXTERNAL

import Html exposing (..)
import Html.Attributes exposing (..)
import Array exposing (Array)
import Dict exposing (Dict)
import Task exposing (Task)
import Time.DateTime as DateTime exposing (DateTime)
import Ansi.Log
import Json.Encode as Encode
import Json.Decode as Decode
import Dom.Scroll as Scroll


-- MODEL


type alias Model =
    { outputStreams : OutputStreams
    , autoScrollMessages : Bool
    }


type alias OutputStreams =
    Dict String BuildStepOutput


type alias BuildStepOutput =
    { taskStep : ProjectTask.Step
    , buildStep : BuildStep
    , streams : List OutputStream
    }


type alias OutputStream =
    { buildStream : BuildStream
    , ansi : Ansi.Log.Model
    , raw : Dict Int BuildStreamOutput
    }


init :
    Context
    -> ProjectTask.Task
    -> Maybe AuthToken
    -> Build
    -> Task Request.Errors.HttpError Model
init context task maybeAuthToken build =
    let
        initialModel outputStreams =
            { outputStreams = outputStreams
            , autoScrollMessages = True
            }
    in
        build
            |> loadBuildStreams context task maybeAuthToken
            |> Task.map initialModel


loadBuildStreams :
    Context
    -> ProjectTask.Task
    -> Maybe AuthToken
    -> Build
    -> Task Request.Errors.HttpError OutputStreams
loadBuildStreams context task maybeAuthToken build =
    build.steps
        |> List.sortBy .number
        |> List.map
            (\buildStep ->
                let
                    maybeTaskStep =
                        task.steps
                            |> Array.fromList
                            |> Array.get buildStep.number
                in
                    ( maybeTaskStep, buildStep )
            )
        |> List.map
            (\( maybeTaskStep, buildStep ) ->
                List.map
                    (\buildStream ->
                        Request.Build.streamOutput context maybeAuthToken buildStream.id
                            |> Task.map (\output -> ( buildStream, maybeTaskStep, buildStep, output ))
                    )
                    buildStep.streams
            )
        |> List.foldl (++) []
        |> Task.sequence
        |> Task.map
            (List.foldr
                (\( buildStream, maybeTaskStep, buildStep, outputStreams ) dict ->
                    case maybeTaskStep of
                        Just taskStep ->
                            let
                                ansiInit =
                                    Ansi.Log.init Ansi.Log.Cooked

                                lineAnsi outputLine ansi =
                                    Ansi.Log.update outputLine.output ansi

                                ansi =
                                    Array.foldl lineAnsi ansiInit outputStreams

                                dictKey =
                                    BuildStep.idToString buildStep.id

                                raw =
                                    Array.foldl (\v a -> Dict.insert v.line v a) Dict.empty outputStreams

                                outputStream =
                                    { buildStream = buildStream
                                    , ansi = ansi
                                    , raw = raw
                                    }
                            in
                                case Dict.get dictKey dict of
                                    Just exists ->
                                        let
                                            updated =
                                                { exists | streams = exists.streams ++ [ outputStream ] }
                                        in
                                            Dict.insert dictKey updated dict

                                    Nothing ->
                                        let
                                            buildStepOutput =
                                                { taskStep = taskStep
                                                , buildStep = buildStep
                                                , streams = [ outputStream ]
                                                }
                                        in
                                            Dict.insert dictKey buildStepOutput dict

                        Nothing ->
                            dict
                )
                Dict.empty
            )



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions _ =
    scrolledToBottom


scrolledToBottom : Sub Msg
scrolledToBottom =
    Decode.decodeValue Decode.bool
        >> Result.toMaybe
        >> Maybe.withDefault False
        |> Ports.onScrolledToBottom
        |> Sub.map ScrolledToBottom



-- CHANNELS --


streamChannelName : BuildStream -> String
streamChannelName stream =
    "stream:" ++ (BuildStream.idToString stream.id)


events : Model -> Dict String (List ( String, Encode.Value -> Msg ))
events model =
    let
        foldStreamEvents ( buildStepId, streams ) dict =
            streams
                |> List.foldl
                    (\stream acc ->
                        let
                            events =
                                [ ( "streamLine:new", AddStreamOutput buildStepId stream ) ]
                        in
                            Dict.insert (streamChannelName stream) events acc
                    )
                    dict
    in
        model.outputStreams
            |> Dict.foldl (\buildStepId val acc -> ( buildStepId, List.map .buildStream val.streams ) :: acc) []
            |> List.foldl foldStreamEvents Dict.empty


leaveChannels : Model -> List String
leaveChannels model =
    Dict.keys (events model)



-- UPDATE --


type Msg
    = AddStreamOutput String BuildStream Encode.Value
    | ScrolledToBottom Bool
    | NoOp


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        AddStreamOutput buildStepId buildStream outputJson ->
            let
                outputStreams =
                    addStreamOutput ( buildStepId, buildStream, outputJson ) model.outputStreams

                scrollCmd =
                    if model.autoScrollMessages then
                        Task.attempt (always NoOp) (Scroll.toBottom "scroll-id")
                    else
                        Cmd.none
            in
                { model | outputStreams = outputStreams }
                    => scrollCmd

        ScrolledToBottom isScrolled ->
            { model | autoScrollMessages = isScrolled }
                => Cmd.none

        NoOp ->
            model => Cmd.none



-- THERE MUST BE AN ERROR HERE... WITH THE BUILD STEP ID?


addStreamOutput : ( String, BuildStream, Encode.Value ) -> OutputStreams -> OutputStreams
addStreamOutput ( buildStepId, targetBuildStream, outputJson ) outputStreams =
    let
        updateOutputStream newBuildOutput =
            let
                debugBuildStepId =
                    Debug.log "DEBUG - BUILD STEP ID" buildStepId

                debugAllBuildStepIds =
                    Debug.log "DEBUG - BUILD ALL BUILD STEP IDS" (Dict.keys outputStreams)
            in
                outputStreams
                    |> Dict.update buildStepId
                        (Maybe.map
                            (\value ->
                                let
                                    debugSuccess =
                                        Debug.log "DEBUG - BUILD SUCCESLLY FOUND" buildStepId

                                    streams =
                                        value.streams
                                            |> List.map
                                                (\stream ->
                                                    if stream.buildStream.id == targetBuildStream.id then
                                                        { stream
                                                            | ansi = Ansi.Log.update newBuildOutput.output stream.ansi
                                                            , raw = Dict.insert newBuildOutput.line newBuildOutput stream.raw
                                                        }
                                                    else
                                                        stream
                                                )
                                in
                                    { value | streams = streams }
                            )
                        )
    in
        outputJson
            |> Decode.decodeValue BuildStream.outputDecoder
            |> Result.toMaybe
            |> Maybe.map
                (\s ->
                    let
                        debugSuccess =
                            Debug.log "DEBUG -" s
                    in
                        updateOutputStream s
                )
            |> Maybe.withDefault
                (let
                    debugErr =
                        Debug.log "DEBUG - BUILD FAIL" outputJson
                 in
                    outputStreams
                )



-- VIEW


view : Build -> Model -> Html Msg
view build { outputStreams } =
    let
        ansiOutput =
            outputStreams
                |> Dict.toList
                |> List.sortBy (\( _, outputStream ) -> outputStream.buildStep.number)
                |> List.map (viewStepContainer build)
    in
        div [] (viewBuildInformation build :: ansiOutput)


viewStepContainer : Build -> ( a, { b | buildStep : BuildStep, streams : List OutputStream, taskStep : Step } ) -> Html Msg
viewStepContainer build ( stepId, { taskStep, buildStep, streams } ) =
    let
        buildStep_ =
            build.steps
                |> List.filter (\s -> s.id == buildStep.id)
                |> List.head
    in
        case buildStep_ of
            Just step ->
                div
                    [ class "card mt-3 b-0"
                    , classList (buildStepBorderColourClassList step)
                    ]
                    [ h5
                        [ class "card-header d-flex justify-content-between"
                        , classList (headerBackgroundColourClassList step)
                        ]
                        [ text (viewCardTitle taskStep)
                        , text " "
                        , viewBuildStepStatusIcon step
                        ]
                    , div [ class "card-body p-0 small b-0" ] [ viewStepLog streams ]
                    ]

            Nothing ->
                text ""


type alias AnsiOutputLine =
    { timestamp : DateTime
    , rawTimestamp : String
    , streamName : String
    , ansiLine : Ansi.Log.Line
    , streamIndex : Int
    , lineNumber : Int
    }


viewStepLog : List OutputStream -> Html Msg
viewStepLog streams =
    streams
        |> flattenStreams
        |> List.map viewLine
        |> table [ class "table-sm mb-0" ]


flattenStreams : List OutputStream -> List AnsiOutputLine
flattenStreams streams =
    streams
        |> List.indexedMap mapStream
        |> List.foldl (++) []
        |> List.sortWith sortAnsiLogLines


mapStream : Int -> OutputStream -> List AnsiOutputLine
mapStream streamIndex { ansi, buildStream, raw } =
    raw
        |> Dict.toList
        |> List.filterMap
            (\( lineNumber, { timestamp, rawTimestamp } ) ->
                ansi
                    |> .lines
                    |> Array.get (lineNumber - 1)
                    |> Maybe.map
                        (\ansiLine ->
                            { timestamp = timestamp
                            , rawTimestamp = rawTimestamp
                            , streamName = buildStream.name
                            , ansiLine = ansiLine
                            , streamIndex = streamIndex
                            , lineNumber = lineNumber
                            }
                        )
            )


sortAnsiLogLines : AnsiOutputLine -> AnsiOutputLine -> Order
sortAnsiLogLines a b =
    if a.streamName == b.streamName then
        Basics.compare a.lineNumber b.lineNumber
    else
        DateTime.compare a.timestamp b.timestamp


viewLine : AnsiOutputLine -> Html Msg
viewLine { timestamp, rawTimestamp, streamName, ansiLine, streamIndex } =
    tr [ class "b-0" ]
        [ td [] [ span [ classList [ "badge" => True, streamBadgeClass streamIndex => True ] ] [ text streamName ] ]
        , td [] [ span [ class "badge badge-light" ] [ text rawTimestamp ] ]
        , td [] [ Ansi.Log.viewLine ansiLine ]
        ]


viewBuildInformation : Build -> Html Msg
viewBuildInformation build =
    let
        dateText date =
            date
                |> Maybe.map formatDateTime
                |> Maybe.withDefault "-"
    in
        div [ class "card mt-3", classList (buildCardClassList build) ]
            [ div [ class "card-body" ]
                [ dl [ class "row mb-0" ]
                    [ dt [ class "col-sm-3" ] [ text "Created" ]
                    , dd [ class "col-sm-9" ] [ text (formatDateTime build.createdAt) ]
                    , dt [ class "col-sm-3" ] [ text "Started" ]
                    , dd [ class "col-sm-9" ] [ text (dateText build.startedAt) ]
                    , dt [ class "col-sm-3" ] [ text "Completed" ]
                    , dd [ class "col-sm-9" ] [ text (dateText build.completedAt) ]
                    , dt [ class "col-sm-3" ] [ text "Status" ]
                    , dd [ class "col-sm-9" ] [ text (Build.statusToString build.status) ]
                    ]
                ]
            ]
