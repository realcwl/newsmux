import grpc
import logging
from concurrent import futures
import deduplicator_pb2
import deduplicator_pb2_grpc
from simhash import Simhash
from nltk.corpus import stopwords
from zhon.hanzi import punctuation
import nltk
import jionlp as jio
import jieba

# Some subsources will have customize prefix, we need to filter them too in
# order to make semantic hashing more accurate.
FILTER_PREFIX = ["【行情】", "【金十图示】", "【提示】", "【股市收盘】"]


class RouteGuideServicer(deduplicator_pb2_grpc.DeduplicatorServicer):
    """Provides methods that implement functionality of deduplication service."""

    def __init__(self):
        pass

    # GetSimHash will return simhash from the input text, based on the specified
    # length. It will tokenize the string and filter all stopwords.
    # This is using the same idea from this legendary paper for web content
    # deduplication:
    # https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/33026.pdf
    def GetSimHash(self, request, context):
        request_text = request.text

        # remove know useless prefix
        for prefix in FILTER_PREFIX:
            if not request_text.startswith(prefix):
                continue
            request_text = request_text.lstrip(prefix)

        words = jieba.lcut(request_text)
        length = request.length

        filtered_words = [
            word for word in words if word not in stopwords.words('english')]
        filtered_words = [word for word in filtered_words if word != " "]
        filtered_words = [
            word for word in filtered_words if word not in punctuation]
        filtered_words = jio.remove_stopwords(filtered_words)

        h = Simhash(filtered_words, f=length).value
        # Remove '0b' in the front, and left pad to specified length
        binary_str = bin(h)[2:].zfill(length)

        return deduplicator_pb2.GetSimHashResponse(binary=binary_str)


def serve():
    ADDR = "[::]:50051"
    nltk.download('stopwords')

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=20))
    deduplicator_pb2_grpc.add_DeduplicatorServicer_to_server(
        RouteGuideServicer(), server)
    server.add_insecure_port(ADDR)
    server.start()
    logging.info("started grpc server at port: " + ADDR)
    server.wait_for_termination()


if __name__ == '__main__':
    logging.basicConfig()
    serve()
