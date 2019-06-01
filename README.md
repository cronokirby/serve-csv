# serve-csv
`serve-csv` is a small CLI program that allows you to start serving a folder
of CSV files as a JSON API. Each CSV file is required to come with a JSON file specifying
how each column maps to a JSON field.

## Example
The example data for this section can be found in the `example-data` folder.

To be able to serve CSV, we need to have some data we want to serve.
Let's put the following rows into `example-data/people.csv`:
```
"John","Smith"
"Alexa","Miller"
"Anthony","Hopkins"
"Maxi","Toshins"
"Clarice","Cromwell"
```

Before we can serve this file, we need to have a corresponding `example-data/people.json`
file, which tells us how to convert each column into a JSON field.
This file looks like this:
```json
{"fields": ["firstName", "lastName"], "types": ["string", "string"]}
```

The `fields` field contains a list of names, one for each column, telling the program
what name to use as a field for that column. The `types` field is an array of the same length
as `fields`, one element per column again, where each element specifies what type the field in that
row should be interpreted as. The available types atm are "string" and "int". "string" will leave the
column as a JSON string, and "int" as a number.

Given this JSON schema, the first row of our CSV will go from:
```
"John","Smith"
```
to
```json
{"firstName": "John", "lastName": "Smith"}
```

To serve this folder as an API, we can do:
```
serve-csv example-data
```

Before starting the API, this will first make sure that each CSV file in the folder has a corresponding
JSON schema. The JSON schemas will also be checked for consistency with the format we aligned above.
Finally, the CSV file will be scanned to check for consistency with the schema as well.

The JSON data for our example should be accessible after the previous command at: `http://localhost:1234/people`.

For each CSV file, the program creates an endpoint to fetch all of the rows, as well as a parameterised endpoint for a single row.
For example, our `people.csv` file maps to a `/people` endpoint, as well as a `/people/i` endpoint for the ith row (starting at 0).
If we had a `cats.csv` file, this would get its own `/cats` and `/cats/i` endpoints as well, etc.

## Usage
```
usage: serve-csv [<flags>] <dir>

Flags:
      --help         Show context-sensitive help (also try --help-long and --help-man).
  -p, --port="1234"  The port to listen on

Args:
  <dir>  The directory to serve
```
