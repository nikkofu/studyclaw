import 'package:pad_app/task_board/api_client.dart';

enum PadApiFeedbackKind { error, infoNotice }

class PadApiFeedback {
  const PadApiFeedback.error(this.message) : kind = PadApiFeedbackKind.error;

  const PadApiFeedback.info(this.message)
      : kind = PadApiFeedbackKind.infoNotice;

  final String message;
  final PadApiFeedbackKind kind;

  bool get isNotice => kind == PadApiFeedbackKind.infoNotice;
}

PadApiFeedback describePadApiFeedback(Object error) {
  if (error is TaskApiException) {
    if (error.errorCode == 'status_unchanged') {
      return PadApiFeedback.info(_describeStatusUnchanged(error));
    }
    return PadApiFeedback.error(_describeTaskApiError(error));
  }
  if (error is FormatException) {
    return const PadApiFeedback.error('服务端返回了无法解析的数据。');
  }
  return PadApiFeedback.error('同步失败：$error');
}

String _describeTaskApiError(TaskApiException error) {
  final details = error.details ?? const <String, dynamic>{};

  switch (error.errorCode) {
    case 'word_list_not_found':
      final date = _detailString(details, 'date');
      if (date != null) {
        return '今天（$date）的默写词单还没准备好。先请家长补充词单，补好后再来同步就能开始默写。';
      }
      return '今天的默写词单还没准备好。先请家长补充词单，补好后再来同步就能开始默写。';
    case 'task_not_found':
      final taskId = _detailInt(details, 'task_id');
      if (taskId != null) {
        return '任务 #$taskId 不存在，可能已被删除或日期已变更。';
      }
      return '当前日期没有可同步的任务，请先刷新任务板。';
    case 'task_group_not_found':
      final subject = _detailString(details, 'subject');
      final groupTitle = _detailString(details, 'group_title');
      if (subject != null && groupTitle != null) {
        return '没有找到“$subject / $groupTitle”对应的任务分组，请先刷新任务板。';
      }
      if (subject != null) {
        return '没有找到“$subject”学科下可同步的任务，请先刷新任务板。';
      }
      return '没有找到可同步的任务分组，请先刷新任务板。';
    case 'missing_required_fields':
      final fields = _detailStringList(details, 'fields');
      if (fields.isNotEmpty) {
        return '缺少必要参数：${fields.map(_fieldLabel).join('、')}。';
      }
      return '缺少必要参数，请检查同步配置后重试。';
    case 'invalid_request_fields':
      final fields = _detailStringList(details, 'fields');
      if (fields.isNotEmpty) {
        return '这些参数缺失或格式不正确：${fields.map(_fieldLabel).join('、')}。';
      }
      return '请求参数缺失或格式不正确，请检查后重试。';
    case 'invalid_query_parameter':
      final field = _detailString(details, 'field');
      if (field != null) {
        return '查询参数“${_fieldLabel(field)}”格式不正确。';
      }
      return '查询参数格式不正确，请检查后重试。';
    case 'invalid_date':
      final field = _detailString(details, 'field');
      if (field != null) {
        return '“${_fieldLabel(field)}”格式无效，请使用 YYYY-MM-DD。';
      }
      return '日期格式无效，请使用 YYYY-MM-DD。';
    case 'invalid_json':
      return '请求体格式无效，请重试。';
    case 'invalid_request':
      return '请求参数无效，请检查同步配置后重试。';
    case 'parser_unavailable':
      return '解析服务暂不可用，请稍后再试。';
    case 'tasks_not_extractable':
      return '任务内容暂时无法解析，请先回到家长端重新确认。';
    case 'internal_error':
      return '服务端处理失败，请稍后再试。';
  }

  if (error.statusCode > 0) {
    return '请求失败（${error.statusCode}）：${error.message}';
  }
  return '网络请求失败：${error.message}';
}

String _describeStatusUnchanged(TaskApiException error) {
  final details = error.details ?? const <String, dynamic>{};
  final statusLabel = _statusLabel(_detailString(details, 'status'));
  final taskId = _detailInt(details, 'task_id');
  final subject = _detailString(details, 'subject');
  final groupTitle = _detailString(details, 'group_title');

  if (taskId != null) {
    return '任务 #$taskId 已经是$statusLabel状态，无需重复同步。';
  }
  if (subject != null && groupTitle != null) {
    return '“$subject / $groupTitle”分组已经是$statusLabel状态，无需重复同步。';
  }
  if (subject != null) {
    return '“$subject”学科任务已经是$statusLabel状态，无需重复同步。';
  }
  return '全部任务已经是$statusLabel状态，无需重复同步。';
}

int? _detailInt(Map<String, dynamic> details, String key) {
  final value = details[key];
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '');
}

String? _detailString(Map<String, dynamic> details, String key) {
  final value = details[key]?.toString().trim();
  if (value == null || value.isEmpty) {
    return null;
  }
  return value;
}

List<String> _detailStringList(Map<String, dynamic> details, String key) {
  final value = details[key];
  if (value is! List) {
    return const <String>[];
  }

  return value
      .map((item) => item?.toString().trim() ?? '')
      .where((item) => item.isNotEmpty)
      .toList();
}

String _fieldLabel(String field) {
  switch (field) {
    case 'subject':
      return '学科';
    case 'group_title':
      return '任务分组';
    case 'task_id':
      return '任务 ID';
    case 'assigned_date':
    case 'date':
      return '任务日期';
    case 'end_date':
      return '结束日期';
    case 'completed':
      return '完成状态';
    default:
      return field;
  }
}

String _statusLabel(String? status) {
  switch (status) {
    case 'completed':
      return '已完成';
    case 'pending':
      return '待完成';
    default:
      return '当前';
  }
}
