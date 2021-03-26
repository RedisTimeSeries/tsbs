import argparse
import json
import os
import re

all_query_types = "cpu-max-all-1,cpu-max-all-8,double-groupby-1,double-groupby-5,double-groupby-all,groupby-orderby-limit,high-cpu-1,high-cpu-all,lastpoint,single-groupby-1-1-1,single-groupby-1-1-12,single-groupby-1-8-1,single-groupby-5-1-1,single-groupby-5-1-12,single-groupby-5-8-1"


def process_txt_files(dirname: str, database: str, queries: [str], prefix: str = ""):
    files_list = os.listdir(dirname)
    p = re.compile('Run complete after (\d+) queries with (\d+) workers.+\(Overall query rate (\d+\.\d*) queries\/sec\):\n(.+)\:\n.+\nall queries.+:\n.+med:\s+(\d+.\d*)ms,.+, count: (\d+)\n')
    results_table = {}
    for query in queries:
        query_table = {}
        for fname in files_list:
            if fname.startswith("result_queries") and database in fname and query in fname and ((prefix != "" and prefix in fname) or (prefix == "")):
                if database not in query_table:
                    query_table[database] = []
                full_path = "{}/{}".format(dirname, fname)

                with open(full_path) as txt_file:
                    result = txt_file.read()
                    match = p.match(result)
                    if match is not None:
                        print(full_path)
                        query_description = match.group(4)
                        number_workers = match.group(2)
                        total_queries = match.group(1)
                        overall_query_rate = match.group(3)
                        overall_query_p50 = match.group(5)
                        result_line = {
                            "query_name": query,
                            "query_description": query_description,
                            "number_workers": number_workers,
                            "total_queries": total_queries,
                            "overall_query_rate": overall_query_rate,
                            "overall_query_p50": overall_query_p50,
                        }
                        query_table[database].append(result_line)
        results_table[query] = query_table
    return results_table


def get_best_results_for(query_results, metric_name, database, query, functor=min):
    metric_value = None
    best_pos = -1
    distinct_results = 0
    if database in query_results:
        database_results = query_results[database]
        total_results_arr = []
        for result in database_results:
            total_results_arr.append(result[metric_name])
        distinct_results = len(total_results_arr)
        if distinct_results:
            metric_value = functor(total_results_arr)
            best_pos = total_results_arr.index(metric_value)
    return metric_value, best_pos, distinct_results


def get_result_at_pos(query_results, metric_name, database, query, pos):
    metric_value = None
    if database in query_results:
        database_results = query_results[database]
        metric_value = database_results[pos][metric_name]
    return metric_value


parser = argparse.ArgumentParser(
    description="Simple script to process TSBS query runners results and output overall metrics",
    formatter_class=argparse.ArgumentDefaultsHelpFormatter,
)
parser.add_argument("--dir", type=str, required=True)
parser.add_argument("--prefix", type=str, default="",
                    help="prefix to filter the result files by")
parser.add_argument("--database", type=str, required=True,
                    help="database to filter the result files by")
parser.add_argument("--query", type=str, default=all_query_types,
                    help="comma separated query types search for results files")

args = parser.parse_args()

results_table = process_txt_files(
    args.dir, args.database, args.query.split(","), args.prefix)
print("-------------------")
print("Database, Query Type, Distinct Runs, Workers, Total queries, p50 latency @rps, rps")
database = args.database
bash_hash_table_of_rps = {}

for query, query_results in results_table.items():
    overall_query_p50_str = "n/a"
    overall_query_rate_str = "n/a"
    number_workers_str = "n/a"
    total_queries_str = "n/a"
    distinct_runs_str = "n/a"
    bash_max_rps = "0"


    overall_query_p50, best_pos, distinct_runs = get_best_results_for(
        query_results, "overall_query_p50", database, query, min)
    if overall_query_p50 is not None:
        overall_query_p50_str = "{}".format(overall_query_p50)
        distinct_runs_str = "{}".format(distinct_runs)
        overall_query_rate = get_result_at_pos(
            query_results, "overall_query_rate", database, query, best_pos)
        if overall_query_rate is not None:
            overall_query_rate_str = "{}".format(overall_query_rate)
            bash_max_rps = "{}".format(int(float(overall_query_rate)))
        number_workers = get_result_at_pos(
            query_results, "number_workers", database, query, best_pos)
        if number_workers is not None:
            number_workers_str = "{}".format(number_workers)
        total_queries = get_result_at_pos(
            query_results, "total_queries", database, query, best_pos)
        if total_queries is not None:
            total_queries_str = "{}".format(total_queries)
    bash_hash_table_of_rps[query] = bash_max_rps
    print("{}, {}, {}, {}, {}, {}, {}".format(database, query, distinct_runs_str,
                                              number_workers_str, total_queries_str, overall_query_p50_str, overall_query_rate_str))
print("-------------------")
print("\n")
print("To run a same-rate competitor benchmark use the following bash RPS_ARRAY:")
bash_array = " ".join( ["\"{}\":\"{}\"".format(k,v) for k,v in bash_hash_table_of_rps.items() ] )
print("RPS_ARRAY=({})\n".format(bash_array))
