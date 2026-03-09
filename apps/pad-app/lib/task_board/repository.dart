import 'package:pad_app/task_board/api_client.dart';
import 'package:pad_app/task_board/models.dart';

abstract interface class TaskBoardRepository {
  Future<TaskBoard> fetchBoard(TaskBoardRequest request);

  Future<TaskBoard> updateSingleTask(
    TaskBoardRequest request, {
    required int taskId,
    required bool completed,
  });

  Future<TaskBoard> updateTaskGroup(
    TaskBoardRequest request, {
    required String subject,
    String? groupTitle,
    required bool completed,
  });

  Future<TaskBoard> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  });
}

class RemoteTaskBoardRepository implements TaskBoardRepository {
  const RemoteTaskBoardRepository({
    this.clientFactory = defaultTaskBoardApiClientFactory,
  });

  final TaskBoardApiClientFactory clientFactory;

  @override
  Future<TaskBoard> fetchBoard(TaskBoardRequest request) async {
    final payload = await _clientFor(request).send(
      'GET',
      '/api/v1/tasks',
      query: {
        'family_id': '${request.familyId}',
        'user_id': '${request.userId}',
        'date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<TaskBoard> updateSingleTask(
    TaskBoardRequest request, {
    required int taskId,
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/item',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'task_id': taskId,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<TaskBoard> updateTaskGroup(
    TaskBoardRequest request, {
    required String subject,
    String? groupTitle,
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/group',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'subject': subject,
        if (groupTitle != null && groupTitle.isNotEmpty)
          'group_title': groupTitle,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  @override
  Future<TaskBoard> updateAllTasks(
    TaskBoardRequest request, {
    required bool completed,
  }) async {
    final payload = await _clientFor(request).send(
      'PATCH',
      '/api/v1/tasks/status/all',
      body: {
        'family_id': request.familyId,
        'assignee_id': request.userId,
        'completed': completed,
        'assigned_date': request.date,
      },
    );
    return TaskBoard.fromJson(payload);
  }

  TaskBoardApiClient _clientFor(TaskBoardRequest request) {
    return clientFactory(request.apiBaseUrl);
  }
}
