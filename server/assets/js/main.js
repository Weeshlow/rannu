(function() {
  'use strict';

  var dataset = $('#dataset');
  var workers = $('#workers');
  //var standardize = $('#standardize');
  var title = $('#title');
  var results = $('#results');
  var loading = $('#loading');
  var chart = $('#scatter');
  var table = {
    creditCard: $('#credit-card-table'),
    iris: $('#iris-table')
  };
  var footer = {
    creditCard: $('#credit-card-footer'),
    iris: $('#iris-footer')
  };

  function populateTable(table, resp) {
    var combined = _.zip(resp.eigenvalues, resp.eigenvectors);
    var sorted = _.sortBy(combined, function(el) { return el[0]; });
    var pc1 = sorted.pop();
    var pc2 = sorted.pop();
    table.find('tbody tr').each(function(i) {
      var row = $(this);
      row.find('.pc1').html(pc1[1][i]);
      row.find('.pc2').html(pc2[1][i]);
    });
  }

  $('#submit').click(function() {
    var numWorkers = workers.val();
    if (!numWorkers) {
      return;
    }
    /*
    var standardized = standardize.val();
    if (!standardize.val()) {
      return;
    }
    */

    results.empty();
    chart.empty();
    loading.show();

    var dataFile;
    switch (dataset.val()) {
    case 'credit-card':
      title.text('Credit Card Defaults');
      $.get('/api/pca/credit-card/' + numWorkers + '/true', function(resp) {
        if (resp.status !== 'ok') {
          alert('Uh oh! ' + resp.message);
          return;
        }
        results.html('<p>Elapsed time: ' + resp.elapsed + ' seconds</p><p>Percent of Variance: ' + resp.percentVariance + '%</p><p>Standardized: Yes</p>');
        //dataFile = standardized === "true" ? 'credit-card-standardized.csv' : 'credit-card.csv';
        dataFile = 'credit-card-standardized.csv';
        scatter('#scatter', '#loading', dataFile, ['No Default', 'Default']);
        $('.table').hide();
        populateTable(table.creditCard, resp);
        table.creditCard.show();
        $('.footer').hide();
        footer.creditCard.show();
      });
      break;
    case 'iris':
      title.text('Iris');
      $.get('/api/pca/iris/' + numWorkers + '/false', function(resp) {
        if (resp.status !== 'ok') {
          console.log(resp);
          alert('Uh oh! ' + resp.message);
          return;
        }
        results.html('<p>Elapsed time: ' + resp.elapsed + ' seconds</p><p>Percent of Variance: ' + resp.percentVariance + '%</p><p>Standardized: No</p>');
        //dataFile = standardized === "true" ? 'iris-standardized.csv' : 'iris.csv';
        dataFile = 'iris.csv';
        scatter('#scatter', '#loading', dataFile, ['Setosa', 'Versicolor', 'Virginica']);
        $('.table').hide();
        populateTable(table.iris, resp);
        table.iris.show();
        $('.footer').hide();
        footer.iris.show();
      });
    default:
      break;
    }
  });
})();
