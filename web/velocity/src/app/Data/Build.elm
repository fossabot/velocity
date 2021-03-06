module Data.Build exposing (..)

import Data.Task as Task exposing (Task)
import Data.BuildStep as BuildStep exposing (BuildStep)
import Data.Commit as Commit exposing (Hash)
import Json.Decode as Decode exposing (Decoder, int, string)
import Json.Decode.Pipeline as Pipeline exposing (custom, decode, hardcoded, required, optional)
import Data.Helpers exposing (stringToDateTime)
import Time.DateTime as DateTime exposing (DateTime)
import UrlParser


type alias Build =
    { id : Id
    , status : Status
    , task : Task
    , steps : List BuildStep
    , createdAt : DateTime
    , completedAt : Maybe DateTime
    , updatedAt : Maybe DateTime
    , startedAt : Maybe DateTime
    }



-- SERIALIZATION --


decoder : Decoder Build
decoder =
    decode Build
        |> required "id" (Decode.map Id string)
        |> required "status" statusDecoder
        |> required "task" Task.decoder
        |> required "buildSteps" (Decode.list BuildStep.decoder)
        |> required "createdAt" stringToDateTime
        |> required "completedAt" (Decode.maybe stringToDateTime)
        |> required "updatedAt" (Decode.maybe stringToDateTime)
        |> required "startedAt" (Decode.maybe stringToDateTime)


statusDecoder : Decoder Status
statusDecoder =
    Decode.string
        |> Decode.andThen
            (\status ->
                case status of
                    "waiting" ->
                        Decode.succeed Waiting

                    "failed" ->
                        Decode.succeed Failed

                    "running" ->
                        Decode.succeed Running

                    "success" ->
                        Decode.succeed Success

                    unknown ->
                        Decode.fail <| "Unknown status: " ++ unknown
            )


statusToString : Status -> String
statusToString status =
    case status of
        Waiting ->
            "waiting"

        Failed ->
            "failed"

        Running ->
            "running"

        Success ->
            "success"


findBuild : List Build -> Id -> Maybe Build
findBuild builds id =
    builds
        |> List.filter (\b -> b.id == id)
        |> List.head


addBuild : List Build -> Build -> List Build
addBuild builds build =
    let
        found =
            List.filter (compare build) builds
    in
        if List.isEmpty found then
            List.append builds [ build ]
        else
            builds



-- IDENTIFIERS --


compare : Build -> Build -> Bool
compare a b =
    idToString a.id == idToString b.id


idParser : UrlParser.Parser (Id -> a) a
idParser =
    UrlParser.custom "ID" (Ok << Id)


idParamHelp : Maybe String -> Maybe Id
idParamHelp maybeValue =
    Maybe.map Id maybeValue


idQueryParser : String -> UrlParser.QueryParser (Maybe Id -> a) a
idQueryParser id =
    UrlParser.customParam id idParamHelp


type Status
    = Waiting
    | Failed
    | Running
    | Success


type Id
    = Id String


idToString : Id -> String
idToString (Id id) =
    id
