var initialized = false;
var name = "";
var tel = "";
var grade = "";
var currentCampus = "";
var currentPeriod = "";
var wantedCampus = "";
var wantedPeriod = "";

function getGrades(gradeSelectObj, gradeValue) {
  gradeSelectObj.html('');

  $.ajax({
        url: '/grades',
        type: 'GET',
        dataType: 'json',
        error : function () {
            alert("获取年级信息失败");
        },   
        success: function (data) {
            if (!data.success) {
                alert("获取年级信息失败:" + data.err_msg);
            } else {
                $.each(data.grades, function(index, value){
                    gradeSelectObj.append('<option value="' + value + '">' + value + '</option>');
                });    
                gradeSelectObj.val(data.grades[0]).change();
                gradeValue = data.grades[0];
		//alert("gradeValue = " + gradeValue); 
            }
        }
  });
}

function getCampuses(grade, campusSelectObj, campusValue) {
  campusSelectObj.html('');

  $.ajax({
        url: '/campuses/' + grade,
        type: 'GET',
        dataType: 'json',
        error : function () {
            alert("获取校区信息失败");
        },
        success: function (data) {
            if (!data.success) {
                alert("获取校区信息失败:" + data.err_msg);
            } else {
                $.each(data.campuses, function(index, value){
                    campusSelectObj.append('<option value="' + value + '">' + value + '</option>');
                });
                campusSelectObj.val(data.campuses[0]).change();
                campusValue = data.campuses[0];
		//alert("campusValue = " + campusValue);
            }
        }
  });
}

function getPeriods(campus, grade, periodSelectObj, periodValue) {
  periodSelectObj.html('');

  $.ajax({
        url: '/periods/' + campus + '/' + grade,
        type: 'GET',
        dataType: 'json',
        error : function () {
            alert("获取时段信息失败." + "校区:" + campus + ",年级:" + grade);
        },
        success: function (data) {
            if (!data.success) {
                alert("获取时段信息失败:" + data.err_msg);
            } else {
                $.each(data.periods, function(index, value){
                    periodSelectObj.append('<option value="' + value + '">' + value + '</option>');
                });
                periodSelectObj.val(data.periods[0]).change();
		periodValue = data.periods[0];
		//alert("periodValue = " + periodValue);
            }
        }
  });
}

$(document).ready(function () {
    //alert("document ready.");

    // Page 1 events.
    $('#name').on("input", function () {
        name = $(this).val();
        tel = $('#tel').val();
        if ((name != "") && (tel != "")) {
            $('#page1-next').removeClass("ui-state-disabled");
        } else {
            $('#page1-next').addClass("ui-state-disabled");
        }
    });

    $('#tel').on("input", function () {
        tel = $(this).val();
        name = $('#name').val();
        if ((name != "") && (tel != "")) {
            $('#page1-next').removeClass("ui-state-disabled");
        } else {
            $('#page1-next').addClass("ui-state-disabled");
        }
    });

    $('#grade').change(function () {
        grade = $(this).find('option:selected').val();

        getCampuses(grade, $('#currentCampus'), currentCampus);
        getCampuses(grade, $('#wantedCampus'), wantedCampus);
    });

    // Page 2 events.
    $('#currentCampus').change(function () {
        currentCampus = $(this).find('option:selected').val();
        getPeriods(currentCampus, grade, $('#currentPeriod'), currentPeriod);
    });

    $('#currentPeriod').change(function () {
        currentPeriod = $(this).find('option:selected').val();
    });

    // Page 3 events.
    $('#wantedCampus').change(function () {
        wantedCampus = $(this).find('option:selected').val();
        getPeriods(wantedCampus, grade, $('#wantedPeriod')), wantedPeriod;
    });

    $('#wantedPeriod').change(function () {
        wantedPeriod = $(this).find('option:selected').val();
    });

    $('#submitBtn').click(function () {
        console.log("name: " + name);
        console.log("tel: " + tel);
        console.log("grade: " + grade);
        console.log("current campus: " + currentCampus);
        console.log("current period: " + currentPeriod);
        console.log("wanted campus: " + wantedCampus);
        console.log("wanted period: " + wantedPeriod);

        postData = {name: name, tel: tel, grade: grade, currentCampus: currentCampus, currentPeriod: currentPeriod, wantedCampus: wantedCampus, wantedPeriod: wantedPeriod};

        $.ajax({
            type: "POST",
            url: "/zb",
            data: postData,
            error: function () {
                alert("提交处理失败.");
            },
            success: function (data) {
                if (data.success) {
                    var msg = "提交成功。请等待学校电话通知处理结果.\n\n";
                    msg += name + ", " + tel + ", " + grade + "\n";
                    msg += "当前: " + currentCampus + ", " + currentPeriod + "\n";
                    msg += "期望: " + wantedCampus + ", " + wantedPeriod + "\n";
                    alert(msg);
                    $.mobile.navigate("/#page4");
                } else {
                    alert("提交失败：" + data.err_msg);
                }
            },
            dataType: "json"
        });

    });

    getGrades($('#grade'), grade);

    initialized = true;

});

$(document).on("pageinit","#page1",function(){
});

$(document).on("pagebeforeshow","#page1",function(){
    //$('#grade').val(grade);
    //alert("page1 before show.");
    name = $('#name').val();
    tel =  $('#tel').val();
    if ((name != "") && (tel != "")) {
        $('#page1-next').removeClass("ui-state-disabled");
    } else {
        $('#page1-next').addClass("ui-state-disabled");
    }
});

$(document).on("pageinit","#page2",function(){
});

$(document).on("pagebeforeshow","#page2",function(){
});

$(document).on("pageinit","#page3",function(){
});

$(document).on("pagebeforeshow","#page3",function(){
});
