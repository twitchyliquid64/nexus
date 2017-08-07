(function () {

    app.filter('bytes', function() {
    	return function(bytes, precision) {
    		if (isNaN(parseFloat(bytes)) || !isFinite(bytes)) return '-';
    		if (typeof precision === 'undefined') precision = 1;
    		var units = ['bytes', 'kB', 'MB', 'GB', 'TB', 'PB'],
    			number = Math.floor(Math.log(bytes) / Math.log(1024));
    		return (bytes / Math.pow(1024, Math.floor(number))).toFixed(precision) +  ' ' + units[number];
    	}
    });

    app.directive("fileinput", [function() {
    return {
      scope: {
        fileinput: "=",
      },
      link: function(scope, element, attributes) {
        element.bind("change", function(changeEvent) {
          console.log("Got fileinput event: ", changeEvent);
          scope.$apply(function(){
            scope.fileinput = changeEvent.target.files[0];
          });
        });
      }
    }
  }]);

    app.directive('uploadModal', function($rootScope, $sce, $http){
      return {
        //scope allows us to setup variable bindings to the parent scope. By default, we share the parent scope. For an isolated one, we should
        //pass an object as the scope attribute which has a dict of the variable name for us, and a string describing where and how to bind it.
        //scope: {color: '@colorAttr'} binds 'color' to the value of color-attr with a one way binding. Only strings supported here: color-attr="{{color}}"
        //scope: {color: '=color'} binds 'color' two way. Pass the object here: color="color".
        scope: {

        },
        //restrict E means its can only be used as an element.
        restrict: 'E',
        templateUrl: function(elem, attr){
          return "/static/views/fs/uploadModal.html?cachebust=3"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.path = '';
          $scope.error = null;

          $rootScope.$on('upload-modal', function(event, args) {
            $scope.open = true;
            $scope.file = null;
            $scope.percent = 0;
            $scope.path = args.path;
          });

          function doUpload() {
            var formData = new FormData();
            formData.append('upload', $scope.file);
            formData.append('path', $scope.path);
            var xhr = new XMLHttpRequest();

            xhr.upload.addEventListener('loadstart', function(evt){
              $scope.uploading = true;
              $scope.error = null;
            }, false);
            xhr.upload.addEventListener('progress', function(evt){
              $scope.$apply(function(){
                $scope.percent = evt.loaded/evt.total*100;                
              });
            }, false);
            //xhr.upload.addEventListener('load', onloadHandler, false);
            xhr.addEventListener('readystatechange', function(evt){
              if (evt.target.readyState == 4) {
                $scope.uploading = false;
                if (evt.target.status == '200'){
                  $scope.open = false;
                  $rootScope.$broadcast('page-change', {page: 'files'});
                } else {
                  $scope.error = evt.target;
                }
              }
            }, false);

            xhr.open('POST', '/web/v1/fs/upload', true);
            xhr.send(formData);
          }

          $scope.upload = function(){
            if (!$scope.file)return;
            doUpload();
          }

          $scope.close = function(){
            $scope.open = false;
          }
        }
      };
  });
})();
