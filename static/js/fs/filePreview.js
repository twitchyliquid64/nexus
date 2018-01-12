app.controller('FilePreviewController', ["$scope", "$rootScope", "$http", function ($scope, $rootScope, $http) {
  $scope.loading = false;
  $scope.error = null;
  $scope.path = null;

  $rootScope.$on('files-preview', function(event, args) {
    $scope.path = args.path;
    $scope.imgsrc = null;
    $scope.mpgsrc = null;
    $scope.pdfsrc = null;
    $scope.file = args.file;
    if (args.file.Name.endsWith(".png") || args.file.Name.endsWith(".jpg") || args.file.Name.endsWith(".gif")){
      $scope.imgsrc = '/web/v1/fs/download' + encodeURIComponent($scope.path);
    }
    if (args.file.Name.endsWith(".mp3")){
      $scope.mpgsrc = '/web/v1/fs/download' + encodeURIComponent($scope.path);
    }
    if (args.file.Name.endsWith(".pdf")){
      $scope.pdfsrc = '/web/v1/fs/download' + encodeURIComponent($scope.path) + '?composition=inline';
    }
  });

  $scope.back = function(){
    $rootScope.$broadcast('files-navigate', {path: $scope.path.substring(0,$scope.path.lastIndexOf("/"))});
    $scope.changePage('files');
  }

  $rootScope.$on('page-change', function(event, args) {
    if (args.page != 'files-preview'){
      $scope.imgsrc = null;
      $scope.mpgsrc = null;
      $scope.pdfsrc = null;
    }
  });
}]);
