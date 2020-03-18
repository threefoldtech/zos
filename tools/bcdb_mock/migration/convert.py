from yaml import load, CLoader
import json
import sys


def main(args):
    if len(args) == 0:
        raise Exception("required file name")
    name = args[0]
    input = open(name)

    data = load(input, CLoader)
    input.close()

    output = open("%s.json" % name, 'w')
    json.dump(data, output)
    output.close()


if __name__ == '__main__':
    main(sys.argv[1:])
