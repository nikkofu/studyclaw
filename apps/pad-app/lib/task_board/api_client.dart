import 'dart:convert';

import 'package:http/http.dart' as http;

typedef TaskBoardApiClientFactory = TaskBoardApiClient Function(String baseUrl);

TaskBoardApiClient defaultTaskBoardApiClientFactory(String baseUrl) {
  return TaskBoardApiClient(baseUrl: baseUrl);
}

class TaskBoardApiClient {
  const TaskBoardApiClient({
    required this.baseUrl,
    this.clientFactory = http.Client.new,
  });

  final String baseUrl;
  final http.Client Function() clientFactory;

  Future<Map<String, dynamic>> send(
    String method,
    String path, {
    Map<String, String>? query,
    Map<String, dynamic>? body,
  }) async {
    final client = clientFactory();
    final uri = Uri.parse(_normalizeBaseUrl(baseUrl))
        .resolve(path)
        .replace(queryParameters: query);

    try {
      final request = http.Request(method, uri);
      request.headers['accept'] = 'application/json';

      if (body != null) {
        request.headers['content-type'] = 'application/json';
        request.body = jsonEncode(body);
      }

      final streamedResponse = await client.send(request);
      final response = await http.Response.fromStream(streamedResponse);
      final payload = _decodePayload(response.body);

      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw TaskApiException(
          message: payload['error']?.toString() ??
              payload['message']?.toString() ??
              '请求失败，状态码 ${response.statusCode}',
          errorCode: payload['error_code']?.toString(),
          details: _decodeDetails(payload['details']),
          uri: uri,
          statusCode: response.statusCode,
        );
      }

      return payload;
    } on TaskApiException {
      rethrow;
    } on FormatException catch (error) {
      throw TaskApiException(
        message: '服务端返回了无法解析的数据: ${error.message}',
        uri: uri,
        statusCode: -1,
      );
    } on http.ClientException catch (error) {
      throw TaskApiException(
        message: error.message,
        uri: uri,
        statusCode: -1,
      );
    } finally {
      client.close();
    }
  }

  Map<String, dynamic> _decodePayload(String responseBody) {
    if (responseBody.isEmpty) {
      return <String, dynamic>{};
    }

    final decoded = jsonDecode(responseBody);
    if (decoded is Map<String, dynamic>) {
      return decoded;
    }
    if (decoded is Map) {
      return decoded.map(
        (key, value) => MapEntry(key.toString(), value),
      );
    }

    throw const FormatException('JSON 根节点不是对象');
  }

  Map<String, dynamic>? _decodeDetails(Object? value) {
    if (value is Map<String, dynamic>) {
      return value;
    }
    if (value is Map) {
      return value.map(
        (key, dynamic item) => MapEntry(key.toString(), item),
      );
    }
    return null;
  }
}

class TaskApiException implements Exception {
  const TaskApiException({
    required this.message,
    this.errorCode,
    this.details,
    required this.uri,
    required this.statusCode,
  });

  final String message;
  final String? errorCode;
  final Map<String, dynamic>? details;
  final Uri uri;
  final int statusCode;

  @override
  String toString() {
    return 'TaskApiException(statusCode: $statusCode, errorCode: $errorCode, uri: $uri, message: $message)';
  }
}

String _normalizeBaseUrl(String baseUrl) {
  final trimmed = baseUrl.trim();
  if (trimmed.endsWith('/')) {
    return trimmed;
  }
  return '$trimmed/';
}
